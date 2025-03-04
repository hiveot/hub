package session

import (
	"bytes"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/wot"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type NotifyType string

// the key under which client session data is stored
const HiveOViewDataKey = "hiveoview"
const dashboardsStorageKey = "dashboards"

const (
	NotifyInfo    NotifyType = "info"
	NotifySuccess NotifyType = "success"
	NotifyError   NotifyType = "error"
	NotifyWarning NotifyType = "warning"
)

type SSEEvent struct {
	Event   string
	Payload string
	ID      string
}

// DefaultExpiryHours TODO: set default expiry in config
const DefaultExpiryHours = 72

// WebClientSession manages the connection and state of a web client session.
type WebClientSession struct {
	// ID of this session
	//sessionID string
	cid string // connection-id as provided by the client

	// the connection ID of this client for correlating requests
	clcid string

	// clientID of the authenticated client
	clientID string

	// Client session data, loaded from the state service
	clientData *SessionData
	// the error is used to retry loading if the client state is requested
	clientStateError error

	// The consumer connection of this session
	co *messaging.Consumer

	// Holder of consumed things for this session
	ctDir *consumedthing.ConsumedThingsDirectory

	// flag, this session is active and can be used to send messages to
	// the hub. If sseChan is set then the return channel is active too.
	isActive atomic.Bool

	// SenderID is the login ID of the user
	//clientID string
	// RemoteAddr of the user
	remoteAddr string

	lastActivity time.Time
	//lastError    error

	// The associated consumer client for pub/sub
	//co *messaging.Consumer

	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channel for calling back to the remote client
	// This is nil if no-one is listening.
	// Only use with mux
	sseChan chan SSEEvent

	// session manager callback
	onClosed func(*WebClientSession)
}

// Consume is short for consumedThingSession.Consume()
func (sess *WebClientSession) Consume(
	thingID string) (ct *consumedthing.ConsumedThing, err error) {
	return sess.ctDir.Consume(thingID)
}

// GetClientID returns the ID of this session's client
func (sess *WebClientSession) GetClientID() string {
	return sess.clientID
}

// GetCLCID returns the client-connection-id of this session
func (sess *WebClientSession) GetCLCID() string {
	return sess.clcid
}

// GetCID returns the connection-id of this session provided by the remote client
func (sess *WebClientSession) GetCID() string {
	return sess.cid
}

// GetClientData returns the hiveoview data model of this client
func (sess *WebClientSession) GetClientData() *SessionData {
	return sess.clientData
}

// GetConsumer returns the hub client connection for use in pub/sub
func (sess *WebClientSession) GetConsumer() *messaging.Consumer {
	return sess.co
}

// GetLastError returns the most recent error, if any
func (sess *WebClientSession) GetLastError() error {
	return sess.clientStateError
}

// GetConsumedThingsDirectory returns the directory of consumed things of this client
func (sess *WebClientSession) GetConsumedThingsDirectory() *consumedthing.ConsumedThingsDirectory {
	return sess.ctDir
}

// GetViewModel returns the hiveoview view model of this client
//func (sess *WebClientSession) GetViewModel() *ClientViewModel {
//	return sess.viewModel
//}

// HandleWebConnectionClosed is called by the web server when a
// web client sse connection closes.
// Note this locks for writing so run as goroutine.
//
// This closes the SSE channel and removes it.
//
// If after 5 seconds a new SSE session has not been re-established, then
// this will close the hub connection, which in turn will end this session.
//
// Note that refreshing the browser page will cause it to disconnect.
// However, if the browser immediately reconnects using the same cid, then
// this session will continue to operate.
func (sess *WebClientSession) HandleWebConnectionClosed() {
	slog.Debug("HandleWebConnectionClosed - web client disconnected",
		slog.String("ClientID", sess.GetClientID()),
		slog.String("cid", sess.cid),
		slog.String("remoteAddr", sess.remoteAddr),
	)

	// Two options.
	// A: F5 should create a new cid (connection-id), ... as this is a new connection (duh)
	// B: Use the same CID and don't close the session and hub connection. Pro is that
	//  session data will be available.
	//   A new CID is required for each tab so can it even be re-used?
	//  how to create a unique CID? web client can use header in sse connection

	sess.mux.Lock()
	// cleanup the channel
	if sess.sseChan != nil {
		close(sess.sseChan)
		sess.sseChan = nil
	}
	sess.mux.Unlock()

	//go func() {
	sess.mux.RLock()
	if sess.sseChan == nil {
		// disconnect from the hub. This will call back into 'HandleHubConnectionClosed'
		// which will end the session.
		sess.co.Disconnect()
	}
	sess.mux.RUnlock()
	//}()
}

// HandleHubConnectionClosed is called when the http connection to the hub has closed.
// This will deactivate this session, remove it and notify a callback.
// This will notify the browser client before disconnecting.
// This can happen if the runtime restarts or if the user logs out from the hub.
func (sess *WebClientSession) HandleHubConnectionClosed() {
	slog.Debug("HandleHubConnectionClosed",
		slog.String("ClientID", sess.GetClientID()),
		slog.String("cid", sess.cid),
		slog.String("remoteAddr", sess.remoteAddr),
	)
	if sess.isActive.Swap(false) {

		// notify session manager to remove this session
		if sess.onClosed != nil {
			sess.onClosed(sess)
		}

		// Shutting down this session should first kill the hub so there might
		// still be a web connection. Attempt to notify.
		// TODO: these notifications should be in JS using sse
		sess.SendNotify(NotifyWarning, "", "Disconnected from the Hub")
		sess.SendSSE("connectStatus", "Disconnected from the Hub")

		// this will call back into HandleWebConnectionClosed, which will not do
		// anything since the channel was already cleaned up.
		sess.mux.Lock()
		defer sess.mux.Unlock()
		if sess.sseChan != nil {
			close(sess.sseChan)
		}
		sess.sseChan = nil
	}
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
// This will be deprecated as the session is removed as soon as the hub or browser connection closes.
// Right now it is useful to detect an internal state discrepancy.
func (sess *WebClientSession) IsActive() bool {
	return sess.isActive.Load()
}

// IsConnected returns the 'connected' status of hub connection
func (sess *WebClientSession) IsConnected() bool {
	return sess.co.IsConnected()
}

func (sess *WebClientSession) Logout() {
	// FIXME
	panic("TODO")
}

// NewSseChan creates a new SSE return channel.
// Intended for use by the sse connection handler.
func (sess *WebClientSession) NewSseChan() chan SSEEvent {
	slog.Debug("NewSseChan", "clientID", sess.GetClientID(),
		"cid", sess.cid, "remoteAddr", sess.remoteAddr)
	sess.mux.Lock()
	defer sess.mux.Unlock()
	if !sess.isActive.Load() {
		slog.Error("Adding a SSE channel to an inactive session. This is unexpected.")
		return nil
	}
	if sess.sseChan != nil {
		// this can happen if two client connects with the same cid.
		// Disconnect the first and let the second one take over
		close(sess.sseChan)
		slog.Error("The session already has a return channel. Disconnecting the initial connection.")
	}
	// need 1+ deep so that writing isn't blocked before reading it
	sess.sseChan = make(chan SSEEvent, 1)
	return sess.sseChan
}

// onHubConnectionChange is invoked on hub client disconnect/reconnect
func (sess *WebClientSession) onHubConnectionChange(connected bool, err error, c messaging.IConnection) {
	lastErrText := ""

	slog.Debug("onHubConnectionChange",
		//slog.String("clientID", stat.SenderID),
		slog.Bool("connected", connected),
		slog.String("lastError", lastErrText))

	if connected {
		sess.SendNotify(NotifySuccess, "", "Connection established with the Hub")
	} else if err != nil {
		//  a normal disconnect?
		sess.SendNotify(NotifyWarning, "", "Connection with Hub failed: "+err.Error())
		// notify the client and close session
		sess.HandleHubConnectionClosed()
	} else {
		// notify the client and close session
		sess.HandleHubConnectionClosed()
	}
}

// onResponse notifies SSE clients of incoming notifications from the Hub
// This is intended for notifying the client UI of the update to props or events.
// The consumed thing itself is already updated.
// FIMXE: that no longer works. The consumed thing isn't updated
//
//	 FIX-1: how is the consumed thing supposed to be updated?
//				A: consumer->consumed thing -> webclientsession
//				B: consumer->webclientsession -> consumed thing
//	   ? it should be the one that subscribes
//	 FIX-2: consumed things should be shared between sessions of the same client
//				so that additional browser tabs doesn't require a reload.
func (sess *WebClientSession) onResponse(resp *messaging.ResponseMessage) error {

	//slog.Debug("received notification",
	//	slog.String("operation", resp.Operation),
	//		slog.String("thingID", resp.ThingID),
	//		slog.String("name", resp.Name),
	//		//slog.Any("data", resp.Data),
	//		slog.String("senderID", resp.SenderID),
	//		slog.String("receiver cid", sess.cid),
	//	)
	// update the directory
	_ = sess.ctDir.OnResponse(resp)

	if resp.Operation == wot.OpObserveProperty {
		// Notify the UI of the property value change:
		//    hx-trigger="sse:{{.AffordanceType}}/{{.ThingID}}/{{.Name}}"
		// TODO: can htmx work with the ResponseMessage or InteractionOutput object?
		propID := fmt.Sprintf("%s/%s/%s",
			messaging.AffordanceTypeProperty, resp.ThingID, resp.Name)
		propVal := tputils.DecodeAsString(resp.Output, 0)
		sess.SendSSE(propID, propVal)
		// also notify of a change to updated timestamp
		propID = fmt.Sprintf("%s/%s/%s/updated",
			messaging.AffordanceTypeProperty, resp.ThingID, resp.Name)
		sess.SendSSE(propID, tputils.DecodeAsDatetime(resp.Updated))
	} else if resp.Operation == wot.OpSubscribeEvent {
		// Publish sse event indicating the event affordance or value has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this event:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{$k}}"
		// where $k is the event ID
		eventID := fmt.Sprintf("%s/%s/%s",
			messaging.AffordanceTypeEvent, resp.ThingID, resp.Name)
		sess.SendSSE(eventID, resp.ToString(0))
		eventID = fmt.Sprintf("%s/%s/%s/updated",
			messaging.AffordanceTypeEvent, resp.ThingID, resp.Name)
		sess.SendSSE(eventID, tputils.DecodeAsDatetime(resp.Updated))
	}
	return nil
}

// ReplaceConsumer replaces the hub consumer connection for this client session.
// This closes the old connection and ignores the callback it gives.
func (sess *WebClientSession) ReplaceConsumer(newCo *messaging.Consumer) {
	oldCo := sess.co
	oldCo.Disconnect()
	sess.co = newCo
	newCo.SetConnectHandler(sess.onHubConnectionChange)
	newCo.SetResponseHandler(sess.onResponse)
}

// SendNotify sends a 'notify' event for showing in a toast popup.
// To send an SSE event use SendSSE()
//
// The msgID is optional. If provided then a followup message with the same ID
// will update the toast instead of adding a new one.
//
//	ntype is the toast notification type: "info", "error", "warning"
//	msgID is the notification message ID.
//	text to include in the notification
func (sess *WebClientSession) SendNotify(ntype NotifyType, msgID string, text string) {
	sess.mux.RLock()
	defer sess.mux.RUnlock()
	if sess.sseChan != nil {
		sess.sseChan <- SSEEvent{Event: "notify",
			Payload: string(ntype) + ":" + msgID + ":" + text}
	} else {
		// not neccesarily an error as a notification can be sent after the channel closes
		//slog.Error("SendNotify. SSE channel is closed")
	}
}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to notify the browser of changes.
func (sess *WebClientSession) SendSSE(event string, content string) {

	// the browser htmx sse handler matches 'event' with htmx hx-trigger name
	// as the triggers are based on thingID/name, the event types are
	// really the ID's. While this is fine, it is unfortunately not compatible
	// with HttpSSEClient, which expects the type to contain event|property|action
	// and the ID the thingID/name/...
	sess.mux.RLock()
	defer sess.mux.RUnlock()
	if sess.sseChan != nil {
		sess.sseChan <- SSEEvent{Event: event, Payload: content, ID: event}
	} else {
		// not neccesarily an error as a notification can be sent after the channel closes
		//slog.Error("SendSSE. SSE channel is closed")
	}
}

// WriteError handles reporting of an error in this session
//
// This logs the error, sends a SSE notification.
// If err is nil then write the httpCode result
// If no httpCode is given then this renders an error message
// If an httpCode is given then this returns the status code
func (sess *WebClientSession) WriteError(w http.ResponseWriter, err error, httpCode int) {
	if err == nil {
		w.WriteHeader(httpCode)
		return
	}
	slog.Error(err.Error())
	sess.SendNotify(NotifyError, "", err.Error())

	if httpCode == 0 {
		output := "Oops: " + err.Error()
		// Write returns http OK
		_, _ = w.Write([]byte(output))
		return
	}

	http.Error(w, err.Error(), httpCode)
}

// WritePage writes a rendered html page or reports the error
// Errors are sent to the UI with the notify event.
func (sess *WebClientSession) WritePage(w http.ResponseWriter, buff *bytes.Buffer, err error) {
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
	} else {
		_, _ = buff.WriteTo(w)
	}
}

// NewWebClientSession creates a new webclient session for the given Hub consumer.
//
// A Web Client session is created on the first request from a web browser, after
// establishing a hub connection using its credentials.
//
// At this point there is not yet a browser SSE return channel. A 'cid' will be
// necessary to link this connection with the expected incoming SSE connection.
//
// This session manages both an outgoing hub connection and an incoming web
// browser SSE connection .
//
// This session closes when A) the hub connection closes, or B) more likely, the web
// browser SSE connection closes. This will result in a call to onClosed.
// The call to onClosed takes place within a locked section, so it should never
// call back into the session or a deadlock will occur.
//
// If no SSE connection is established within 3 seconds then this session
// closes itself to avoid being orphaned. Normally this session is cleaned up
// after the SSE connection closes but if it is never established then it
// can hang around indefinitely.
//
//	cid is the web client provided connectionID used to associate http request with SSE clients
//	co is the consumer connected to the Hub
//	remoteAddr is the web client remote address
//	configBucket to store dashboards. This will be closed when this session is removed.
//	onClose is the callback to invoke when this session is closed.
func NewWebClientSession(
	cid string, co *messaging.Consumer, remoteAddr string,
	configBucket buckets.IBucket,
	onClosed func(*WebClientSession)) *WebClientSession {
	var err error

	// Each web client session has their own connection to the Hub through
	// a consumed things directory.
	//
	// The consumed things directory holds the consumed thing instances for use
	// by the web client. Consumed things are automatically updated when Thing
	// subscription updates are received.
	coDir := consumedthing.NewConsumedThingsDirectory(co)

	webSess := WebClientSession{
		cid:          cid,
		clcid:        co.GetClientID() + "-" + cid,
		clientID:     co.GetClientID(),
		remoteAddr:   remoteAddr,
		lastActivity: time.Now(),
		clientData:   NewClientDataModel(configBucket),
		//viewModel:    NewClientViewModel(hc),
		co:       co,
		ctDir:    coDir,
		onClosed: onClosed,
	}
	co.SetConnectHandler(webSess.onHubConnectionChange)

	// onResponse is called with async responses to requests
	co.SetResponseHandler(webSess.onResponse)

	webSess.isActive.Store(co.IsConnected())

	// TODO: selectively subscribe instead of everything, but, based on what?
	err = co.Subscribe("", "")
	err = co.ObserveProperty("", "")
	if err != nil {
		//webSess.lastError = err
	}

	// prevent orphaned sessions. Cleanup after 3 sec
	// the number is arbitrary and not sensitive.
	go func() {
		time.Sleep(time.Second * 30) // for testing change from 3 to 30
		webSess.mux.RLock()
		hasSSE := webSess.sseChan != nil
		webSess.mux.RUnlock()

		if !hasSSE && webSess.IsActive() {
			slog.Info("Removing orphaned web-session (no sse connection) within 3 seconds", "cid", webSess.cid)
			webSess.co.Disconnect()
		}
	}()

	return &webSess
}
