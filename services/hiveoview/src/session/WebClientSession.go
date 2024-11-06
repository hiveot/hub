package session

import (
	"bytes"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/state/stateclient"
	"github.com/hiveot/hub/wot/consumedthing"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type NotifyType string

// the key under which client session data is stored
const HiveOViewDataKey = "hiveoview"

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

	// Client session data, loaded from the state service
	clientData *ClientDataModel
	// the error is used to retry loading if the client state is requested
	clientStateError error

	// Holder of consumed things for this session
	// FIXME: if a tdd is not found initially then reload it
	cts *consumedthing.ConsumedThingsDirectory

	// flag, this session is active and can be used to send messages to
	// the hub. If sseChan is set then the return channel is active too.
	isActive atomic.Bool

	// SenderID is the login ID of the user
	//clientID string
	// RemoteAddr of the user
	remoteAddr string

	lastActivity time.Time
	lastError    error

	// The associated hub client for pub/sub
	hc hubclient.IConsumerClient
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
	return sess.cts.Consume(thingID)
}

// GetClientID returns the ID of this session's client
func (sess *WebClientSession) GetClientID() string {
	return sess.hc.GetClientID()
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
func (sess *WebClientSession) GetClientData() *ClientDataModel {
	// if loading previously failed then recover
	sess.mux.RLock()
	clientStateError := sess.clientStateError
	sess.mux.RUnlock()
	if clientStateError != nil {
		err := sess.LoadState()
		_ = err
	}
	// return something
	return sess.clientData
}

// GetHubClient returns the hub client connection for use in pub/sub
func (sess *WebClientSession) GetHubClient() hubclient.IConsumerClient {
	return sess.hc
}

// GetConsumedThingsDirectory returns the directory of consumed things of this client
func (sess *WebClientSession) GetConsumedThingsDirectory() *consumedthing.ConsumedThingsDirectory {
	return sess.cts
}

// GetViewModel returns the hiveoview view model of this client
//func (sess *WebClientSession) GetViewModel() *ClientViewModel {
//	return sess.viewModel
//}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (sess *WebClientSession) GetStatus() (bool, string, error) {
	return sess.hc.GetConnectionStatus()
}

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
		sess.hc.Disconnect()
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

		// if a web connection is still there then attempt to notify
		sess.SendNotify(NotifyWarning, "Disconnected from the Hub")

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

// LoadState loads the client session state containing dashboard and other model data,
// and clear 'clientModelChanged' status
func (sess *WebClientSession) LoadState() error {
	// load the stored view state from the state service
	stateCl := stateclient.NewStateClient(sess.hc)
	clientData := NewClientDataModel()
	found, err := stateCl.Get(HiveOViewDataKey, &clientData)
	_ = found

	// then lock and load
	sess.mux.Lock()
	defer sess.mux.Unlock()
	if err != nil {
		sess.lastError = err
		//slog.Error("LoadState failed", "err", err.Error())
		return err
	} else {
		sess.clientData = clientData
	}
	return nil
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
	sess.sseChan = make(chan SSEEvent, 0)
	return sess.sseChan
}

// onHubConnectionChange is invoked on hub client disconnect/reconnect
func (sess *WebClientSession) onHubConnectionChange(connected bool, err error) {
	lastErrText := ""

	slog.Debug("onHubConnectionChange",
		//slog.String("clientID", stat.SenderID),
		slog.Bool("connected", connected),
		slog.String("lastError", lastErrText))

	if connected {
		sess.SendNotify(NotifySuccess, "Connection established with the Hub")
	} else if err != nil {
		//  a normal disconnect?
		sess.SendNotify(NotifyWarning, "Connection with Hub failed: "+err.Error())
		// notify the client and close session
		sess.HandleHubConnectionClosed()
	} else {
		// notify the client and close session
		sess.HandleHubConnectionClosed()
	}
}

// onMessage notifies SSE clients of incoming messages from the Hub
// This is intended for notifying the client UI of the update to props, events or actions.
// The consumed thing itself is already updated.
func (sess *WebClientSession) onMessage(msg *hubclient.ThingMessage) {

	slog.Debug("received message",
		slog.String("type", msg.MessageType),
		slog.String("thingID", msg.ThingID),
		slog.String("name", msg.Name),
		//slog.Any("data", msg.Data),
		slog.String("senderID", msg.SenderID),
		slog.String("receiver cid", sess.cid),
		slog.String("messageID", msg.MessageID),
	)
	if msg.MessageType == vocab.MessageTypeProperty {
		// Publish a sse event for each property
		// The UI that displays this event can use this as a trigger to load the
		// property value:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{k}}"
		// notify the browser both of the property value and the timestamp
		thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, msg.Name)
		propVal := utils.DecodeAsString(msg.Data)
		sess.SendSSE(thingAddr, propVal)
		thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Name)
		sess.SendSSE(thingAddr, msg.GetUpdated())
	} else if msg.MessageType == vocab.MessageTypeProgressUpdate {
		// report unhandled delivery updates
		// for now just pass it to the notification toaster
		stat := hubclient.ActionProgress{}
		_ = utils.DecodeAsObject(msg.Data, &stat)

		// TODO: figure out a way to replace the existing notification if the messageID
		//  is the same (status changes from applied to delivered)
		if stat.Error != "" {
			sess.SendNotify(NotifyError, stat.Error)
		} else if stat.Progress == vocab.ProgressStatusCompleted {
			sess.SendNotify(NotifySuccess, "Action successful")
		} else {
			sess.SendNotify(NotifyWarning, "Action delivery: "+stat.Progress)
		}
		//} else if msg.MessageType == vocab.MessageTypeEvent &&
		//	msg.ThingID == digitwin.DirectoryDThingID &&
		//	msg.Name == digitwin.DirectoryEventThingUpdated {

		//// Update the TD in the consumed-thing if it exists
		//cts := sess.GetConsumedThingsDirectory()
		//td := cts.UpdateTD(msg.Data)
		//
		//// Publish sse event to the UI to update their TD.
		//// The UI that displays this event can use this as a trigger to reload the
		//// fragment that displays this TD:
		////    hx-trigger="sse:{{.Thing.ThingID}}"
		//thingAddr := td.ID
		//sess.SendSSE(thingAddr, "")
	} else {
		// Publish sse event indicating the event affordance or value has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this event:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{$k}}"
		// where $k is the event ID
		eventName := fmt.Sprintf("%s/%s", msg.ThingID, msg.Name)
		sess.SendSSE(eventName, msg.DataAsText())
		eventName = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Name)
		sess.SendSSE(eventName, msg.GetUpdated())
	}
}

// ReplaceConnection replaces the hub connection in this session.
// This closes the old connection and ignores the callback it gives.
func (sess *WebClientSession) ReplaceConnection(hc hubclient.IConsumerClient) {
	oldHC := sess.hc
	sess.hc = hc
	hc.SetConnectHandler(sess.onHubConnectionChange)
	//hc.SetMessageHandler(sess.onMessage)
	oldHC.SetConnectHandler(nil)
	oldHC.SetMessageHandler(nil)
	oldHC.Disconnect()
}

// SaveState stores the current client session model using the state service,
// if 'clientModelChanged' is set.
//
// This returns an error if the state service is not reachable.
func (sess *WebClientSession) SaveState() error {
	if !sess.clientData.Changed() {
		return nil
	}
	sess.mux.RLock()
	clientState := sess.clientData
	sess.mux.RUnlock()

	stateCl := stateclient.NewStateClient(sess.GetHubClient())
	err := stateCl.Set(HiveOViewDataKey, &clientState)
	if err != nil {
		sess.lastError = err
		return err
	}
	sess.clientData.SetChanged(false)
	return err
}

// SendNotify sends a 'notify' event for showing in a toast popup.
// To send an SSE event use SendSSE()
//
//	ntype is the toast notification type: "info", "error", "warning"
func (sess *WebClientSession) SendNotify(ntype NotifyType, text string) {
	sess.mux.RLock()
	defer sess.mux.RUnlock()
	if sess.sseChan != nil {
		sess.sseChan <- SSEEvent{Event: "notify", Payload: string(ntype) + ":" + text}
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
		// FIXME: the channel gets blocked, which should never happen
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
	sess.SendNotify(NotifyError, err.Error())

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

// NewWebClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
//
// A Web Client session is created on the first http request with a valid token, or
// valid login. After a hub connection is established, this session can be created.
// At this point there is not yet an SSE return channel.
// The cid will be neccesary to link this connection with the expected incoming SSE connection.
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
//	hc is the establihsed hub connection
//	remoteAddr is the web client remote address
//	onClose is the callback to invoke when this session is closed.
func NewWebClientSession(
	cid string, hc hubclient.IConsumerClient, remoteAddr string, noState bool,
	onClosed func(*WebClientSession)) *WebClientSession {
	var err error

	cs := WebClientSession{
		cid:          cid,
		clcid:        hc.GetClientID() + "-" + cid,
		remoteAddr:   remoteAddr,
		hc:           hc,
		lastActivity: time.Now(),
		clientData:   NewClientDataModel(),
		//viewModel:    NewClientViewModel(hc),
		cts:      consumedthing.NewConsumedThingsSession(hc),
		onClosed: onClosed,
	}
	//hc.SetMessageHandler(cs.onMessage)
	hc.SetConnectHandler(cs.onHubConnectionChange)
	// this is a bit quirky but its a transition period
	cs.cts.SetEventHandler(cs.onMessage)

	isConnected, _, _ := hc.GetConnectionStatus()
	cs.isActive.Store(isConnected)

	// restore the session data model
	if !noState {
		err = cs.LoadState()
		if err != nil {
			slog.Error("unable to load client state from state service",
				"clientID", cs.hc.GetClientID(), "err", err.Error())
			cs.SendNotify(NotifyWarning, "Unable to restore session: "+err.Error())
			cs.lastError = err
		}
	}

	// TODO: selectively subscribe instead of everything, but, based on what?
	err = hc.Subscribe("", "")
	err = hc.Observe("", "")
	if err != nil {
		cs.lastError = err
	}

	// prevent orphaned sessions. Cleanup after 3 sec
	// the number is arbitrary and not sensitive.
	go func() {
		time.Sleep(time.Second * 3)
		cs.mux.RLock()
		hasSSE := cs.sseChan != nil
		cs.mux.RUnlock()

		if !hasSSE && cs.IsActive() {
			slog.Info("Removing orphaned web-session (no sse connection) after 3 seconds", "cid", cs.cid)
			cs.hc.Disconnect()
		}
	}()

	return &cs
}
