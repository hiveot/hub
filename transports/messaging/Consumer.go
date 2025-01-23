package messaging

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// Consumer provides the messaging functionality for consumers
// This provides a golang API to consumer operations.
type Consumer struct {
	// The underlying transport connection for delivering and receiving requests and responses
	cc transports.IConnection

	// The timeout to use when waiting for a response
	rpcTimeout time.Duration

	// Request and Response channel helper
	rnrChan *RnRChan

	// application callback for reporting connection status change
	appConnectHandler transports.ConnectionHandler

	// application callback that handles asynchronous responses
	appResponseHandler transports.ResponseHandler
}

// Disconnect the client.
func (co *Consumer) Disconnect() {
	co.cc.Disconnect()
}

// GetClientID returns the client's account ID
func (co *Consumer) GetClientID() string {
	return co.cc.GetClientID()
}

// GetConnection returns the connection of this consumer
func (co *Consumer) GetConnection() transports.IConnection {
	return co.cc
}

// InvokeAction invokes an action on a thing and wait for the response
func (co *Consumer) InvokeAction(
	dThingID, name string, input any, output any) error {

	req := transports.NewRequestMessage(
		wot.OpInvokeAction, dThingID, name, input, "")
	resp, err := co.SendRequest(req, true)

	if err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}
	if output != nil && resp.Output != nil {
		err = tputils.Decode(resp.Output, output)
	}
	return err
}

// IsConnected returns true if the consumer has a connection
func (co *Consumer) IsConnected() bool {
	return co.cc.IsConnected()
}

// Logout requests invalidating all client sessions.
func (co *Consumer) Logout() (err error) {

	slog.Info("Logout",
		slog.String("clientID", co.GetClientID()))

	req := transports.NewRequestMessage(wot.HTOpLogout, "", "", nil, "")
	_, err = co.SendRequest(req, true)
	return err
}

// ObserveProperty sends a request to observe one or all properties
func (co *Consumer) ObserveProperty(thingID string, name string) error {
	op := wot.OpObserveProperty
	if name == "" {
		op = wot.OpObserveAllProperties
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
	resp, err := co.SendRequest(req, true)
	_ = resp
	return err
}

// websocket connection status handler
func (co *Consumer) onConnect(connected bool, err error, c transports.IConnection) {
	if co.appConnectHandler != nil {
		co.appConnectHandler(connected, err, c)
	}
}

// onResponse passes a response to the RnR response channel and falls back to pass
// it to the registered application response handler. If neither is available
// then turn the response in a notification and pass it to the notification handler.
func (co *Consumer) onResponse(resp *transports.ResponseMessage) error {

	handled := co.rnrChan.HandleResponse(resp)
	if handled {
		return nil
	}

	// handle the response as an async response with no wait handler registered
	if co.appResponseHandler == nil {
		if resp.Status == transports.StatusPending {
			// NOTE: if no response is expected then this could be an out-of-order response
			// instead of receiving 'pending' 'completed', the completed response is
			// received first.
			// Ignore the pending response for now.
			return nil
		} else {
			// at least one of the handlers should be registered
			slog.Error("Response received but no handler registered",
				"operation", resp.Operation,
				"clientID", co.GetClientID(),
				"thingID", resp.ThingID,
				"name", resp.Name,
				"correlationID", resp.CorrelationID)
			err := fmt.Errorf("Response received but no handler registered")
			return err
		}
	}
	return co.appResponseHandler(resp)
}

// Ping the server and wait for a pong response
func (co *Consumer) Ping() error {
	correlationID := shortid.MustGenerate()
	req := transports.NewRequestMessage(wot.HTOpPing, "", "", "", correlationID)
	resp, err := co.SendRequest(req, true)
	if err != nil {
		return err
	}
	if resp.Output == nil {
		return errors.New("ping returned successfully but received no data")
	}
	return nil
}

// ReadAllEvents sends a request to read all Thing event values
// This is not a WoT operation (but maybe it should be)
func (co *Consumer) ReadAllEvents(thingID string) (values map[string]any, err error) {
	err = co.Rpc(wot.HTOpReadAllEvents, thingID, "", nil, &values)
	return values, err
}

// ReadAllProperties sends a request to read all Thing property values
func (co *Consumer) ReadAllProperties(thingID string) (values map[string]any, err error) {
	err = co.Rpc(wot.OpReadAllProperties, thingID, "", nil, &values)
	return values, err
}

// ReadAllTDs sends a request to read all TDs from an agent
// This returns an array of TDs in JSON format
// This is not a WoT operation (but maybe it should be)
func (co *Consumer) ReadAllTDs() (tdJSONs []string, err error) {
	err = co.Rpc(wot.HTOpReadAllTDs, "", "", nil, &tdJSONs)
	return tdJSONs, err
}

// ReadEvent sends a request to read a Thing event value.
// This returns the value as described in the TD event affordance schema.
// This is not a WoT operation (but maybe it should be)
func (co *Consumer) ReadEvent(thingID, name string) (value any, err error) {
	err = co.Rpc(wot.HTOpReadEvent, thingID, name, nil, &value)
	return value, err
}

// ReadProperty sends a request to read a Thing property value
// This returns the value as described in the TD property affordance schema.
func (co *Consumer) ReadProperty(thingID, name string) (value any, err error) {
	err = co.Rpc(wot.OpReadProperty, thingID, name, nil, &value)
	return value, err
}

// ReadTD sends a request to read the latest Thing TD
// This returns the TD in JSON format.
// This is not a WoT operation (but maybe it should be)
func (co *Consumer) ReadTD(thingID string) (tdJSON string, err error) {
	err = co.Rpc(wot.HTOpReadTD, thingID, "", nil, &tdJSON)
	return tdJSON, err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (co *Consumer) RefreshToken(oldToken string) (newToken string, err error) {

	// FIXME: what is the WoT standard for refreshing a token using http?
	slog.Info("RefreshToken",
		slog.String("clientID", co.GetClientID()))

	req := transports.NewRequestMessage(wot.HTOpRefresh, "", "", oldToken, "")
	resp, err := co.SendRequest(req, true)

	// set the new token as the bearer token
	if err == nil {
		newToken = tputils.DecodeAsString(resp.Output, 0)
	}
	return newToken, err
}

// Rpc sends a request message and waits for a response.
// This returns an error if the request fails or if the response contains an error
func (co *Consumer) Rpc(operation, thingID, name string, input any, output any) error {
	correlationID := shortid.MustGenerate()
	req := transports.NewRequestMessage(operation, thingID, name, input, correlationID)
	resp, err := co.SendRequest(req, true)
	if err == nil {
		if resp.Status == transports.StatusFailed {
			detail := fmt.Sprintf("%v", resp.Output)
			errTxt := resp.Error
			if detail != "" {
				errTxt += "\n" + detail
			}
			err = errors.New(errTxt)
		} else if resp.Output != nil && output != nil {
			err = tputils.Decode(resp.Output, output)
		}
	}
	return err
}

// SendRequest sends an operation request and optionally waits for completion or timeout.
// If waitForCompletion is true and no correlationID is provided then a correlationID will
// be generated to wait for completion.
// If waitForCompletion is false then the response will go to the response handler
func (co *Consumer) SendRequest(req *transports.RequestMessage, waitForCompletion bool) (
	resp *transports.ResponseMessage, err error) {

	t0 := time.Now()
	slog.Info("SendRequest",
		slog.String("op", req.Operation),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID),
	)
	// if not waiting then return asap with a pending response
	if !waitForCompletion {
		err = co.cc.SendRequest(req)
		resp = req.CreateResponse(nil, err)
		resp.Status = transports.StatusPending
		return resp, err
	}

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// open a return channel for the response
	rChan := co.rnrChan.Open(req.CorrelationID)

	err = co.cc.SendRequest(req)

	if err != nil {
		slog.Warn("SendRequest: failed sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
		co.rnrChan.Close(req.CorrelationID)
		return resp, err
	}
	// hmm, not pretty but during login the connection status can be ignored
	// the alternative is not to use SendRequest but plain TLS post
	ignoreDisconnect := req.Operation == wot.HTOpLogin || req.Operation == wot.HTOpRefresh

	resp, err = co.WaitForCompletion(rChan, req.Operation, req.CorrelationID, ignoreDisconnect)

	t1 := time.Now()
	duration := t1.Sub(t0)
	if err != nil {
		slog.Info("SendRequest: failed",
			slog.String("op", req.Operation),
			slog.Int64("duration msec", duration.Milliseconds()),
			slog.String("correlationID", req.CorrelationID),
			slog.String("error", err.Error()))
	} else {
		slog.Debug("SendRequest: success",
			slog.String("op", req.Operation),
			slog.Float64("duration msec", float64(duration.Microseconds())/1000),
			slog.String("correlationID", req.CorrelationID))
	}
	return resp, err
}

//
//// SetConnectHandler sets the notification handler of connection failure
//// Intended to notify the client that a reconnect or relogin is needed.
//func (cl *Consumer) SetConnectHandler(cb func(connected bool, err error)) {
//	cl.mux.Lock()
//	cl.AppConnectHandler = cb
//	cl.mux.Unlock()
//}
//
//// SetNotificationHandler set the handler that receives server notifications
//func (cl *BaseClient) SetNotificationHandler(cb transports.NotificationHandler) {
//	cl.BaseMux.Lock()
//	cl.appNotificationHandler = cb
//	cl.BaseMux.Unlock()
//}

//// Set the form lookup handler
//func (cl *BaseClient) SetGetForm(getForm transports.GetFormHandler) {
//	cl.BaseGetForm = getForm
//}

// SetResponseHandler set the handler that receives asynchronous responses
// Those are response to requests that are not waited for using the baseRnR handler.
func (co *Consumer) SetResponseHandler(cb transports.ResponseHandler) {
	co.appResponseHandler = cb
}

// Subscribe to one or all events of a thing
// name is the event to subscribe to or "" for all events
func (co *Consumer) Subscribe(thingID string, name string) error {
	op := wot.OpSubscribeEvent
	if name == "" {
		op = wot.OpSubscribeAllEvents
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
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
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
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
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
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
	rChan chan *transports.ResponseMessage, operation, correlationID string, ignoreDisconnect bool) (
	resp *transports.ResponseMessage, err error) {

	waitCount := 0
	var completed bool
	var hasResponse bool

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
		hasResponse, resp = co.rnrChan.WaitForResponse(rChan, time.Second)
		if hasResponse {
			// ignore pending or other transient responses
			completed = resp.Status == transports.StatusCompleted ||
				resp.Status == transports.StatusFailed
		}
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
	} else if resp.Error != "" {
		// if response data holds an error type then return that as the error
		err = errors.New(resp.Error)
	}
	return resp, err
}

// WriteProperty is a helper to send a write property request
func (co *Consumer) WriteProperty(thingID string, name string, input any, wait bool) error {
	correlationID := shortid.MustGenerate()
	req := transports.NewRequestMessage(wot.OpWriteProperty, thingID, name, input, correlationID)
	resp, err := co.SendRequest(req, wait)
	_ = resp
	return err
}

// NewAgent creates a new consumer instance for sending requests and receiving responses.
//
// Consumers connect to a Thing server as a client.
//
// This is a wrapper around the ClientConnection that provides WoT response messages
// publishing properties and events to subscribers and publishing a TD.

// NewConsumer returns a new instance of the WoT consumer for use with the given connection.
// This provides the API for common WoT operations by creating requests and waiting for responses.
//
//	cc the client connection to use for sending requests and receiving responses
//	respHandler callback for passing unhandled/async responses
//	connHandler callback when connection status changes.
func NewConsumer(cc transports.IConnection,
	respHandler transports.ResponseHandler,
	connHandler transports.ConnectionHandler,
	timeout time.Duration) *Consumer {

	consumer := Consumer{
		cc:                 cc,
		rnrChan:            NewRnRChan(),
		appConnectHandler:  connHandler,
		appResponseHandler: respHandler,
		rpcTimeout:         timeout,
	}
	cc.SetResponseHandler(consumer.onResponse)
	cc.SetConnectHandler(consumer.onConnect)
	return &consumer
}
