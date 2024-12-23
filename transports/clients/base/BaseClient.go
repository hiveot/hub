package base

import (
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// BaseClient provides base functionality of all transport clients
type BaseClient struct {
	// ID of this client
	BaseClientID string
	// unique connectionID start with the clientID
	BaseConnectionID string

	// CA certificate to verify the server with
	BaseCaCert *x509.Certificate
	// RPC timeout
	BaseTimeout time.Duration
	// protected operations
	BaseMux sync.RWMutex

	// The full server's URL schema://host:port/path
	BaseFullURL string
	// The server host:port
	BaseHostPort string
	// protocol ProtocolTypeHTTPS/SSESC/MQTT/WSS
	BaseProtocolType string

	// client is connected
	BaseIsConnected atomic.Bool

	// Request and Response channel helper
	BaseRnrChan *tputils.RnRChan

	// BasePubRequest is set to the implementation of publishing a
	// request by the subclass without waiting for a response.
	// Workaround for golang not supporting inheritance.
	BasePubRequest func(msg transports.RequestMessage) error

	// application callback for reporting connection status change
	AppConnectHandler func(connected bool, err error)

	// application callback that receives consumer facing messages from the hub
	appNotificationHandler transports.NotificationHandler

	// application callback that receives asynchronous responses
	appResponseHandler transports.ResponseHandler

	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests. (agents)
	appRequestHandler transports.RequestHandler
}

// GetClientID returns the client's account ID
func (cl *BaseClient) GetClientID() string {
	return cl.BaseClientID
}

// GetConnectionID returns the client's connection ID
func (cl *BaseClient) GetConnectionID() string {
	return cl.BaseConnectionID
}

// GetProtocolType returns the type of protocol this client supports
func (cl *BaseClient) GetProtocolType() string {
	return cl.BaseProtocolType
}

// GetServerURL returns the schema://address:port/path of the server connection
func (cl *BaseClient) GetServerURL() string {
	return cl.BaseFullURL
}

// InvokeAction invokes an action on a thing and wait for the response
// This is a helper that sends a request with operation wot.OpInvokeAction
func (cl *BaseClient) InvokeAction(
	dThingID, name string, input any, output any) error {

	req := transports.NewRequestMessage(wot.OpInvokeAction, dThingID, name, input, "")
	response, err := cl.SendRequest(req, true)

	if err != nil {
		return err
	} else if response.Error != "" {
		return errors.New(response.Error)
	}
	if output != nil && response.Output != nil {
		err = tputils.Decode(response.Output, output)
	}
	return err
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *BaseClient) IsConnected() bool {
	return cl.BaseIsConnected.Load()
}

// Marshal encodes the native data into the wire format
func (cl *BaseClient) Marshal(data any) []byte {
	jsonData, _ := jsoniter.Marshal(data)
	return jsonData
}

// ObserveProperty observe one or all properties
func (cl *BaseClient) ObserveProperty(thingID string, name string) error {
	op := wot.OpObserveProperty
	if name == "" {
		op = wot.OpObserveAllProperties
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
	resp, err := cl.SendRequest(req, false)
	_ = resp
	return err
}

// OnNotification passes a notification to the registered handler or log an error if
// no handler is set.
func (cl *BaseClient) OnNotification(notif transports.NotificationMessage) {
	cl.OnNotification(notif)
	if cl.appNotificationHandler == nil {
		slog.Error("handleSseEvent: Received notification but no handler is set",
			"operation", notif.Operation,
			"thingID", notif.ThingID,
			"name", notif.Name,
		)
	} else {
		cl.appNotificationHandler(notif)
	}
}

// OnRequest passes a request to the application request handler and returns the response.
// Handler must be set by agent subclasses during init.
// This logs an error if no agent handler is set.
func (cl *BaseClient) OnRequest(req transports.RequestMessage) transports.ResponseMessage {

	// handle requests if any
	if cl.appRequestHandler == nil {
		err := fmt.Errorf("Received request but no handler is set")
		resp := req.CreateResponse(transports.StatusFailed, nil, err)
		return resp
	}
	resp := cl.appRequestHandler(req)
	return resp
}

// OnResponse passes a response to the RnR response channel and falls back to pass
// it to the registered application response handler. If neither is available
// then turn the response in a notification and pass it to the notification handler.
func (cl *BaseClient) OnResponse(resp transports.ResponseMessage) {

	handled := cl.BaseRnrChan.HandleResponse(resp)
	if handled {
		return
	}

	// handle the response as an async response with no wait handler registered
	if cl.appResponseHandler != nil {
		cl.appResponseHandler(resp)
		return
	}

	if cl.appNotificationHandler != nil {
		// last resport is to pass to the notification handler
		notif := transports.NewNotificationMessage(
			resp.Operation, resp.ThingID, resp.Name, resp.Output)
		cl.appNotificationHandler(notif)
	} else {
		// at least one of the handlers should be registered
		slog.Error("Response received but no handler registered",
			"operation", resp.Operation,
			"thingID", resp.ThingID,
			"name", resp.Name,
			"requestID", resp.RequestID)
	}
}

// Ping the server and wait for a pong response
func (cl *BaseClient) Ping() error {
	requestID := shortid.MustGenerate()
	req := transports.NewRequestMessage(wot.HTOpPing, "", "", "", requestID)
	resp, err := cl.SendRequest(req, true)
	if err != nil {
		return err
	}
	if resp.Output == nil {
		return errors.New("ping returned successfully but received no data")
	}
	return nil
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *BaseClient) RefreshToken(oldToken string) (newToken string, err error) {

	// FIXME: what is the WoT standard for refreshing a token using http?
	slog.Info("RefreshToken",
		slog.String("clientID", cl.GetClientID()))

	req := transports.NewRequestMessage(wot.HTOpRefresh, "", "", oldToken, "")
	resp, err := cl.SendRequest(req, true)

	// set the new token as the bearer token
	if err == nil {
		newToken = tputils.DecodeAsString(resp.Output)
	}
	return newToken, err
}

// SendRequest sends an operation request and optionally waits for completion or timeout.
// If waitForCompletion is true and no requestID is provided then a requestID will
// be generated to wait for completion.
func (cl *BaseClient) SendRequest(
	req transports.RequestMessage, waitForCompletion bool) (resp transports.ResponseMessage, err error) {

	t0 := time.Now()
	slog.Info("SendRequest",
		slog.String("op", req.Operation),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("requestID", req.RequestID),
	)
	if req.RequestID == "" && waitForCompletion {
		req.RequestID = shortid.MustGenerate()
	}
	// open a return channel for the response
	rChan := cl.BaseRnrChan.Open(req.RequestID)

	err = cl.BasePubRequest(req)

	if err != nil {
		slog.Warn("SendRequest: failed sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"requestID", req.RequestID,
			"err", err.Error())
		cl.BaseRnrChan.Close(req.RequestID)
		return resp, err
	}
	// hmm, not pretty but during login the connection status can be ignored
	// the alternative is not to use SendRequest but plain TLS post
	ignoreDisconnect := req.Operation == wot.HTOpLogin || req.Operation == wot.HTOpRefresh

	resp, err = cl.WaitForResponse(rChan, req.Operation, req.RequestID, ignoreDisconnect)

	t1 := time.Now()
	duration := t1.Sub(t0)
	if err != nil {
		slog.Info("SendRequest: failed",
			slog.String("op", req.Operation),
			slog.Int64("duration msec", duration.Milliseconds()),
			slog.String("requestID", req.RequestID),
			slog.String("error", err.Error()))
	} else {
		slog.Info("SendRequest: success",
			slog.String("op", req.Operation),
			slog.Float64("duration msec", float64(duration.Microseconds())/1000),
			slog.String("requestID", req.RequestID))
	}
	return resp, err
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *BaseClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.BaseMux.Lock()
	cl.AppConnectHandler = cb
	cl.BaseMux.Unlock()
}

// SetNotificationHandler set the handler that receives server notifications
func (cl *BaseClient) SetNotificationHandler(cb transports.NotificationHandler) {
	cl.BaseMux.Lock()
	cl.appNotificationHandler = cb
	cl.BaseMux.Unlock()
}

// SetRequestHandler set the application handler for incoming requests
func (cl *BaseClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.appRequestHandler = cb
}

// SetResponseHandler set the handler that receives asynchronous responses
// Those are response to requests that are not waited for using the baseRnR handler.
func (cl *BaseClient) SetResponseHandler(cb transports.ResponseHandler) {
	cl.BaseMux.Lock()
	cl.appResponseHandler = cb
	cl.BaseMux.Unlock()
}

// Subscribe to one or all events of a thing
// name is the event to subscribe to or "" for all events
func (cl *BaseClient) Subscribe(thingID string, name string) error {
	op := wot.OpSubscribeEvent
	if name == "" {
		op = wot.OpSubscribeAllEvents
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
	resp, err := cl.SendRequest(req, false)
	_ = resp
	return err
}

// Unmarshal decodes the wire format to native data
func (cl *BaseClient) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// UnobserveProperty a previous observed property or all properties
func (cl *BaseClient) UnobserveProperty(thingID string, name string) error {
	op := wot.OpUnobserveProperty
	if name == "" {
		op = wot.OpUnobserveAllProperties
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
	resp, err := cl.SendRequest(req, false)
	_ = resp
	return err
}

// Unsubscribe is a helper for sending an unsubscribe request
func (cl *BaseClient) Unsubscribe(thingID string, name string) error {
	op := wot.OpUnsubscribeEvent
	if name == "" {
		op = wot.OpUnsubscribeAllEvents
	}
	req := transports.NewRequestMessage(op, thingID, name, nil, "")
	resp, err := cl.SendRequest(req, false)
	_ = resp
	return err
}

// WaitForResponse waits for a response message on the given requestID channel,
// or until N seconds passed, or the connection drops.
//
// If a proper response is received it is written to the given output and nil
// (no error) is returned.
// If anything goes wrong, an error is returned
func (cl *BaseClient) WaitForResponse(
	rChan chan transports.ResponseMessage, operation, requestID string, ignoreDisconnect bool) (
	resp transports.ResponseMessage, err error) {

	waitCount := 0
	var completed bool

	for !completed {
		// If the server connection no longer exists then don't wait any longer.
		// The problem with this is that a response can already be available before
		// a disconnect occurred, which we'll miss here.
		// Especially in case of login or token refresh isconnected check should
		// not be used.
		if !cl.IsConnected() && !ignoreDisconnect {
			err = errors.New("connection lost")
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.BaseTimeout {
			err = errors.New("timeout. No response")
			break
		}
		if waitCount > 0 {
			slog.Info("WaitForResponse (wait)",
				slog.Int("count", waitCount),
				slog.String("clientID", cl.GetClientID()),
				slog.String("operation", operation),
				slog.String("requestID", requestID),
			)
		}
		completed, resp = cl.BaseRnrChan.WaitForResponse(rChan, time.Second)
		waitCount++
	}

	// ending the wait
	cl.BaseRnrChan.Close(requestID)
	slog.Debug("WaitForResponse (result)",
		slog.String("clientID", cl.GetClientID()),
		slog.String("operation", operation),
		slog.String("requestID", requestID),
	)

	// check for errors
	if err != nil {
		slog.Warn("WaitForResponse failed", "err", err.Error())
	} else if resp.Error != "" {
		// if response data holds an error type then return that as the error
		err = errors.New(resp.Error)
	}
	return resp, err
}

// WriteProperty is a helper to send a write property request
func (cl *BaseClient) WriteProperty(thingID string, name string, input any, wait bool) error {
	requestID := shortid.MustGenerate()
	req := transports.NewRequestMessage(wot.OpWriteProperty, thingID, name, input, requestID)
	resp, err := cl.SendRequest(req, wait)
	_ = resp
	return err
}
