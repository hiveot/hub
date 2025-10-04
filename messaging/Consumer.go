package messaging

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hub/wot"
	"github.com/teris-io/shortid"
)

const DefaultRpcTimeout = time.Second * 60 // 60 for testing; 3 seconds

// Consumer provides the messaging functionality for consumers
// This provides a golang API to consumer operations.
type Consumer struct {
	// application callback for reporting connection status change
	appConnectHandlerPtr atomic.Pointer[ConnectionHandler]

	// application callback that handles asynchronous responses
	appResponseHandlerPtr atomic.Pointer[ResponseHandler]

	// application callback that handles notifications
	appNotificationHandlerPtr atomic.Pointer[NotificationHandler]

	// The underlying transport connection for delivering and receiving requests and responses
	cc IConnection

	mux sync.RWMutex

	// The timeout to use when waiting for a response
	rpcTimeout time.Duration

	// Request and Response channel helper
	rnrChan *RnRChan
}

// Disconnect the client connection.
// Do not use this consumer after disconnect.
func (co *Consumer) Disconnect() {
	co.cc.Disconnect()
	// the connect callback is still needed to notify the client of a disconnect
}

// GetClientID returns the client's account ID
func (co *Consumer) GetClientID() string {
	cinfo := co.cc.GetConnectionInfo()
	return cinfo.ClientID
}

// GetConnection returns the underlying connection of this consumer
func (co *Consumer) GetConnection() IConnection {
	return co.cc
}

// InvokeAction invokes an action on a thing and wait for the response
// If the response type is known then provide it with output, otherwise use interface{}
func (co *Consumer) InvokeAction(
	dThingID, name string, input any, output any) error {

	req := NewRequestMessage(wot.OpInvokeAction, dThingID, name, input, "")
	resp, err := co.SendRequest(req, true)

	if err != nil {
		return err
	} else if resp.Error != nil {
		return resp.Error.AsError()
	}
	err = resp.Decode(output)
	return err
}

// IsConnected returns true if the consumer has a connection
func (co *Consumer) IsConnected() bool {
	return co.cc.IsConnected()
}

// Logout requests invalidating all client sessions.
//func (co *Consumer) Logout() (err error) {
//
//	slog.Info("Logout",
//		slog.String("clientID", co.GetClientID()))
//
//	req := transports.NewRequestMessage(wot.HTOpLogout, "", "", nil, "")
//	_, err = co.SendRequest(req, true)
//	return err
//}

// ObserveProperty sends a request to observe one or all properties
//
//	thingID is empty for all things
//	name is empty for all properties of the selected things
func (co *Consumer) ObserveProperty(thingID string, name string) error {
	op := wot.OpObserveProperty
	if name == "" {
		op = wot.OpObserveAllProperties
	}
	req := NewRequestMessage(op, thingID, name, nil, "")
	resp, err := co.SendRequest(req, true)
	_ = resp
	return err
}

// connection status handler
func (co *Consumer) onConnect(connected bool, err error, c IConnection) {
	hPtr := co.appConnectHandlerPtr.Load()
	if hPtr != nil {
		(*hPtr)(connected, err, c)
	}
}

// onNotification passes a response to the RnR response channel and falls back to pass
// it to the registered application response handler. If neither is available
// then turn the response in a notification and pass it to the notification handler.
func (co *Consumer) onNotification(notif *NotificationMessage) {

	hPtr := co.appNotificationHandlerPtr.Load()
	if hPtr == nil {
		if notif.Operation == wot.OpInvokeAction {
			// not everyone is interested in action progress updates
			slog.Info("onNotification: Action progress received. No handler registered",
				"operation", notif.Operation,
				"clientID", co.GetClientID(),
				"thingID", notif.ThingID,
				"name", notif.Name,
			)
		} else {
			// When subscribing then a handler is expected
			slog.Error("onNotification: Notification received but no handler registered",
				"correlationID", notif.CorrelationID,
				"operation", notif.Operation,
				"clientID", co.GetClientID(),
				"thingID", notif.ThingID,
				"name", notif.Name,
			)
		}
		return
	}
	// pass the response to the registered handler
	slog.Info("onNotification",
		"operation", notif.Operation,
		"clientID", co.GetClientID(),
		"thingID", notif.ThingID,
		"name", notif.Name,
		"value", notif.ToString(50),
	)
	(*hPtr)(notif)
}

// onResponse passes a response to the RnR response channel and falls back to pass
// it to the registered application response handler. If neither is available
// then turn the response in a notification and pass it to the notification handler.
func (co *Consumer) onResponse(resp *ResponseMessage) error {

	handled := co.rnrChan.HandleResponse(resp)
	if handled {
		return nil
	}

	// handle the response as an async response with no wait handler registered
	hPtr := co.appResponseHandlerPtr.Load()
	if hPtr == nil {
		// at least one of the handlers should be registered
		slog.Error("Response received but no handler registered",
			"correlationID", resp.CorrelationID,
			"operation", resp.Operation,
			"clientID", co.GetClientID(),
			"thingID", resp.ThingID,
			"name", resp.Name,
		)
		err := fmt.Errorf("response received but no handler registered")
		return err
	}
	// pass the response to the registered handler
	slog.Info("onResponse (async)",
		"operation", resp.Operation,
		"clientID", co.GetClientID(),
		"thingID", resp.ThingID,
		"name", resp.Name,
		"value", resp.ToString(50),
	)
	return (*hPtr)(resp)
}

// Ping the server and wait for a pong response
// This uses the underlying transport native method of ping-pong.
func (co *Consumer) Ping() error {
	correlationID := shortid.MustGenerate()
	req := NewRequestMessage(wot.HTOpPing, "", "", nil, correlationID)
	resp, err := co.SendRequest(req, true)
	if err != nil {
		return err
	}
	if resp.Value == nil {
		return errors.New("ping returned successfully but received no data")
	}
	return nil
}

// QueryAction obtains the status of an action
//
// Q: http-basic protocol returns an array per action in QueryAllActions but only
//
//	a single action in QueryAction. This is inconsistent.
//
// The underlying protocol binding constructs the ActionStatus from the
// protocol specific messages.
// The hiveot protocol passes this as-is as the output.
func (co *Consumer) QueryAction(thingID, name string) (
	value ActionStatus, err error) {

	err = co.Rpc(wot.OpQueryAction, thingID, name, nil, &value)
	// if state is empty then this action has not run before
	if err == nil && value.State == "" {
		value.ThingID = thingID
		value.Name = name
	}
	return value, err
}

// QueryAllActions returns a map of action status for all actions of a thing.
//
// This returns a map of actionName and the last known action status.
//
// Q: http-basic protocol returns an array for each action. What is the use-case?
//
//	that can have multiple concurrent actions? An actuator can only move in
//	one direction at the same time.
//	Maybe the array only applies to stateless actions?
//
// This depends on the underlying protocol binding to construct appropriate
// ActionStatus message. All hiveot protocols include full information.
// WoT bindings might not include update timestamp and such.
func (co *Consumer) QueryAllActions(thingID string) (
	values map[string]ActionStatus, err error) {

	err = co.Rpc(wot.OpQueryAllActions, thingID, "", nil, &values)
	return values, err
}

// ReadAllEvents sends a request to read all Thing event values from the hub.
//
// This returns a map of eventName and the last received event message.
//
// TODO: maybe better to send the last events on subscription...
//func (co *Consumer) ReadAllEvents(thingID string) (
//	values map[string]transports.ThingValue, err error) {
//
//	err = co.Rpc(wot.HTOpReadAllEvents, thingID, "", nil, &values)
//	return values, err
//}

// ReadAllProperties sends a request to read all Thing property values.
//
// This depends on the underlying protocol binding to construct appropriate
// ResponseMessages and include information such as Timestamp. All hiveot protocols
// include full information. WoT bindings might be more limited.
func (co *Consumer) ReadAllProperties(thingID string) (
	values map[string]ThingValue, err error) {

	err = co.Rpc(wot.OpReadAllProperties, thingID, "", nil, &values)
	return values, err
}

// ReadAllTDs sends a request to read all TDs from an agent
// This returns an array of TDs in JSON format
// This is not a WoT operation (but maybe it should be)
//func (co *Consumer) ReadAllTDs() (tdJSONs []string, err error) {
//	err = co.Rpc(wot.HTOpReadAllTDs, "", "", nil, &tdJSONs)
//	return tdJSONs, err
//}

// ReadEvent sends a request to read a Thing event value.
//
// This returns a ResponseMessage containing the value as described in the TD
// event affordance schema.
//
// TODO: maybe better to send the last events on subscription...
//func (co *Consumer) ReadEvent(thingID, name string) (
//	value transports.ThingValue, err error) {
//
//	err = co.Rpc(wot.HTOpReadEvent, thingID, name, nil, &value)
//	return value, err
//}

// ReadProperty sends a request to read a Thing property value.
//
// This depends on the underlying protocol binding to construct appropriate
// ResponseMessages and include information such as Timestamp. All hiveot protocols
// include full information. WoT bindings might be too limited.
func (co *Consumer) ReadProperty(thingID, name string) (
	value ThingValue, err error) {

	err = co.Rpc(wot.OpReadProperty, thingID, name, nil, &value)
	return value, err
}

// RetrieveThing sends a request to read the latest Thing TD
// This returns the TD in JSON format.
// This is not a WoT operation (but maybe it should be)
//func (co *Consumer) RetrieveThing(thingID string) (tdJSON string, err error) {
//	err = co.Rpc(wot.HTOpReadTD, thingID, "", nil, &tdJSON)
//	return tdJSON, err
//}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'SetBearerToken'
// This is specific to the Hiveot Hub.
//func (co *Consumer) RefreshToken(oldToken string) (newToken string, err error) {
//
//	// FIXME: what is the WoT standard for refreshing a token using http?
//	slog.Info("RefreshToken",
//		slog.String("clientID", co.GetClientID()))
//
//	req := transports.NewRequestMessage(wot.HTOpRefresh, "", "", oldToken, "")
//	resp, err := co.SendRequest(req, true)
//
//	// set the new token as the bearer token
//	if err == nil {
//		newToken = tputils.DecodeAsString(resp.Value, 0)
//	}
//	return newToken, err
//}

// Rpc sends a request message and waits for a response.
// This returns an error if the request fails or if the response contains an error
func (co *Consumer) Rpc(operation, thingID, name string, input any, output any) error {
	correlationID := shortid.MustGenerate()
	req := NewRequestMessage(operation, thingID, name, input, correlationID)
	resp, err := co.SendRequest(req, true)
	if err == nil {
		if resp.Error != nil {
			err = resp.Error.AsError()
		} else {
			err = resp.Decode(output)
		}
	}
	return err
}

// SendRequest sends an operation request and optionally waits for completion or timeout.
// If waitForCompletion is true and no correlationID is provided then a correlationID will
// be generated to wait for completion.
//
// If waitForCompletion is false then any response will go to the async response
// handler and this returns nil response.
// If waitForCompletion is true this will wait until a response is received with
// a matching correlationID, or until a timeout occurs.
//
// If the request has no correlation ID, one will be generated.
func (co *Consumer) SendRequest(req *RequestMessage, waitForCompletion bool) (
	resp *ResponseMessage, err error) {

	t0 := time.Now()
	slog.Info("SendRequest: ->",
		slog.String("op", req.Operation),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID),
		slog.String("input", req.ToString(30)),
	)
	// if not waiting then return asap with a pending response
	if !waitForCompletion {
		err = co.cc.SendRequest(req)
		return nil, err
	}

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// open a return channel for the response
	rChan := co.rnrChan.Open(req.CorrelationID)

	err = co.cc.SendRequest(req)

	if err != nil {
		slog.Warn("SendRequest ->: error in sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
		co.rnrChan.Close(req.CorrelationID)
		return resp, err
	}
	// hmm, not pretty but during login the connection status can be ignored
	// the alternative is not to use SendRequest but plain TLS post
	//ignoreDisconnect := req.Operation == wot.HTOpLogin || req.Operation == wot.HTOpRefresh
	ignoreDisconnect := false
	resp, err = co.WaitForCompletion(rChan, req.Operation, req.CorrelationID, ignoreDisconnect)

	t1 := time.Now()
	duration := t1.Sub(t0)
	if err != nil {
		slog.Info("SendRequest: <- failed",
			slog.String("op", req.Operation),
			slog.Int64("duration msec", duration.Milliseconds()),
			slog.String("correlationID", req.CorrelationID),
			slog.String("error", err.Error()))
	} else {
		errMsg := ""
		if resp.Error != nil {
			errMsg = resp.Error.String()
		}
		slog.Info("SendRequest: <-",
			slog.String("op", req.Operation),
			slog.Float64("duration msec", float64(duration.Microseconds())/1000),
			slog.String("correlationID", req.CorrelationID),
			slog.String("err", errMsg),
			slog.String("output", resp.ToString(30)),
		)
	}
	return resp, err
}

// SetConnectHandler sets the notification handler of changes to this consumer connection
// Intended to notify the client that a reconnect or relogin is needed.
// Only a single handler is supported. This replaces the previously set callback.
func (co *Consumer) SetConnectHandler(cb ConnectionHandler) {
	if cb == nil {
		co.appConnectHandlerPtr.Store(nil)
	} else {
		co.appConnectHandlerPtr.Store(&cb)
	}
}

// SetNotificationHandler sets the notification handler of events for this consumer
// Only a single handler is supported. This replaces the previously set callback.
func (co *Consumer) SetNotificationHandler(cb NotificationHandler) {
	if cb == nil {
		co.appNotificationHandlerPtr.Store(nil)
	} else {
		co.appNotificationHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the handler that receives asynchronous responses
// Those are response to requests that are not waited for using the baseRnR handler.
func (co *Consumer) SetResponseHandler(cb ResponseHandler) {
	if cb == nil {
		co.appResponseHandlerPtr.Store(nil)
	} else {
		co.appResponseHandlerPtr.Store(&cb)
	}
}

// Subscribe to one or all events of a thing
// name is the event to subscribe to or "" for all events
func (co *Consumer) Subscribe(thingID string, name string) error {
	op := wot.OpSubscribeEvent
	if name == "" {
		op = wot.OpSubscribeAllEvents
	}
	req := NewRequestMessage(op, thingID, name, nil, "")
	resp, err := co.SendRequest(req, true)
	_ = resp
	return err
}

// UnobserveProperty a previous observed property or all properties
func (co *Consumer) UnobserveProperty(thingID string, name string) error {
	op := wot.OpUnobserveProperty
	if name == "" {
		op = wot.OpUnobserveAllProperties
	}
	req := NewRequestMessage(op, thingID, name, nil, "")
	resp, err := co.SendRequest(req, true)
	_ = resp
	return err
}

// Unsubscribe is a helper for sending an unsubscribe request
func (co *Consumer) Unsubscribe(thingID string, name string) error {
	op := wot.OpUnsubscribeEvent
	if name == "" {
		op = wot.OpUnsubscribeAllEvents
	}
	req := NewRequestMessage(op, thingID, name, nil, "")
	resp, err := co.SendRequest(req, true)
	_ = resp
	return err
}

// WaitForCompletion waits for a completed or failed response message on the
// given correlationID channel, or until N seconds passed, or the connection drops.
//
// If a proper response is received it is written to the given output and nil
// (no error) is returned.
// If anything goes wrong, an error is returned
func (co *Consumer) WaitForCompletion(
	rChan chan *ResponseMessage, operation, correlationID string, ignoreDisconnect bool) (
	resp *ResponseMessage, err error) {

	waitCount := 0
	var completed bool

	for !completed {
		// If the server connection no longer exists then don't wait any longer.
		// The problem with this is that a response can already be available before
		// a disconnect occurred, which we'll miss here.
		// Especially in case of login or token refresh isconnected check should
		// not be used.
		if !co.cc.IsConnected() && !ignoreDisconnect {
			err = errors.New("connection lost")
			break
		}

		// wait at most co.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > co.rpcTimeout {
			err = errors.New("timeout. No response")
			break
		}
		if waitCount > 0 {
			slog.Info("WaitForCompletion (wait)",
				slog.Int("count", waitCount),
				slog.String("clientID", co.GetClientID()),
				slog.String("operation", operation),
				slog.String("correlationID", correlationID),
			)
		}
		completed, resp = co.rnrChan.WaitForResponse(rChan, time.Second)
		waitCount++
	}

	// ending the wait
	co.rnrChan.Close(correlationID)
	slog.Debug("WaitForCompletion (result)",
		slog.String("clientID", co.GetClientID()),
		slog.String("operation", operation),
		slog.String("correlationID", correlationID),
	)

	// check for errors
	if err != nil {
		slog.Warn("WaitForCompletion failed", "err", err.Error())
	} else if resp == nil {
		err = fmt.Errorf("no response received on request '%s'", operation)
	} else if resp.Error != nil {
		// if response data holds an error type then return that as the error
		err = resp.Error.AsError()
	}
	return resp, err
}

// WriteProperty is a helper to send a write property request
func (co *Consumer) WriteProperty(thingID string, name string, input any, wait bool) error {
	correlationID := shortid.MustGenerate()
	req := NewRequestMessage(wot.OpWriteProperty, thingID, name, input, correlationID)
	resp, err := co.SendRequest(req, wait)
	_ = resp
	return err
}

// NewConsumer returns a new instance of the WoT consumer for use with the given
// connection. The connection should not be used by others as this consumer takes
// possession by registering connection callbacks.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetNotificationHandler to set the callback to receive async notifications.
// Use SetResponseHandler to set the callback to receive async responses.
// Use SetConnectHandler to set the callback to be notified of connection changes.
//
//	cc the client connection to use for sending requests and receiving responses.
//	timeout of the rpc connections or 0 for default (3 sec)
func NewConsumer(cc IConnection, rpcTimeout time.Duration) *Consumer {
	if rpcTimeout == 0 {
		rpcTimeout = DefaultRpcTimeout
	}
	consumer := Consumer{
		cc:         cc,
		rnrChan:    NewRnRChan(),
		rpcTimeout: rpcTimeout,
	}
	consumer.SetNotificationHandler(nil)
	consumer.SetConnectHandler(nil)
	consumer.SetResponseHandler(nil)
	// set the connection callbacks to this consumer
	cc.SetNotificationHandler(consumer.onNotification)
	cc.SetResponseHandler(consumer.onResponse)
	cc.SetConnectHandler(consumer.onConnect)
	return &consumer
}
