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
}

// DefaultExpiryHours TODO: set default expiry in config
const DefaultExpiryHours = 72

// WebClientSession manages the connection and state of a web client session.
type WebClientSession struct {
	// ID of this session
	sessionID string

	// Client session data, loaded from the state service
	clientState *ClientDataModel

	// Client view model for generating presentation data
	viewModel *ClientViewModel

	// Holder of consumed things for this session
	cts *consumedthing.ConsumedThingsSession

	// ClientID is the login ID of the user
	//clientID string
	// RemoteAddr of the user
	remoteAddr string

	lastActivity time.Time
	lastError    error

	// The associated hub client for pub/sub
	hc hubclient.IConsumerClient
	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channels for this session
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent
}

//func (cs *WebClientSession) AddSSEClient(c chan SSEEvent) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	cs.sseClients = append(cs.sseClients, c)
//
//	go func() {
//		if cs.IsActive() {
//			//cs.SendNotify(NotifySuccess, "Connected to the Hub")
//		} else {
//			cs.SendNotify(NotifyError, "Not connected to the Hub")
//		}
//	}()
//}

// Close the session.
// This closes the hub connection and SSE data channels
func (wcs *WebClientSession) Close() {
	wcs.mux.Lock()
	for _, sseChan := range wcs.sseClients {
		close(sseChan)
	}
	wcs.sseClients = nil
	wcs.mux.Unlock()
	wcs.hc.Disconnect()
}

// CloseSSEChan closes a previously created SSE channel and removes it.
func (wcs *WebClientSession) CloseSSEChan(c chan SSEEvent) {
	slog.Debug("DeleteSSEChan channel", "clientID", wcs.hc.ClientID())
	wcs.mux.Lock()
	defer wcs.mux.Unlock()
	if wcs.sseClients != nil {
		for i, sseClient := range wcs.sseClients {
			if sseClient == c {
				// FIXME: sometimes, when using the debugger, the channel is already closed.
				wcs.sseClients = append(wcs.sseClients[:i], wcs.sseClients[i+1:]...)
				close(c)
				break
			}
		}
	}
}

// Consume is short for consumedThingSession.Consume()
func (wcs *WebClientSession) Consume(
	thingID string) (ct *consumedthing.ConsumedThing, err error) {

	return wcs.cts.Consume(thingID)
}

// CreateSSEChan creates a new SSE channel to communicate with.
// The channel has a buffer of 1 to allow sending a ping message on connect.
// Call CloseSSEClient to close and clean up
func (wcs *WebClientSession) CreateSSEChan() chan SSEEvent {
	wcs.mux.Lock()
	defer wcs.mux.Unlock()
	sseChan := make(chan SSEEvent, 1)
	wcs.sseClients = append(wcs.sseClients, sseChan)
	return sseChan
}

// GetClientData returns the hiveoview data model of this client
func (wcs *WebClientSession) GetClientData() *ClientDataModel {
	return wcs.clientState
}

// GetHubClient returns the hub client connection for use in pub/sub
func (wcs *WebClientSession) GetHubClient() hubclient.IConsumerClient {
	return wcs.hc
}

// GetConsumedThingsSession returns the consumed things model of this client
func (wcs *WebClientSession) GetConsumedThingsSession() *consumedthing.ConsumedThingsSession {
	return wcs.cts
}

// GetViewModel returns the hiveoview view model of this client
func (wcs *WebClientSession) GetViewModel() *ClientViewModel {
	return wcs.viewModel
}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (wcs *WebClientSession) GetStatus() hubclient.TransportStatus {
	status := wcs.hc.GetStatus()
	return status
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
func (wcs *WebClientSession) IsActive() bool {
	status := wcs.hc.GetStatus()
	return status.ConnectionStatus == hubclient.Connected ||
		status.ConnectionStatus == hubclient.Connecting
}

// LoadState loads the client session state containing dashboard and other model data,
// and clear 'clientModelChanged' status
func (wcs *WebClientSession) LoadState() error {
	stateCl := stateclient.NewStateClient(wcs.hc)

	wcs.clientState = NewClientDataModel()
	found, err := stateCl.Get(HiveOViewDataKey, &wcs.clientState)
	_ = found
	if err != nil {
		wcs.lastError = err
		return err
	}
	return nil
}

// onConnectChange is invoked on disconnect/reconnect
func (wcs *WebClientSession) onConnectChange(stat hubclient.TransportStatus) {
	lastErrText := ""
	if stat.LastError != nil {
		lastErrText = stat.LastError.Error()
	}
	slog.Info("onConnectChange",
		slog.String("clientID", stat.ClientID),
		slog.String("status", string(stat.ConnectionStatus)),
		slog.String("lastError", lastErrText))

	if stat.ConnectionStatus == hubclient.Connected {
		wcs.SendNotify(NotifySuccess, "Connection established with the Hub")
	} else if stat.ConnectionStatus == hubclient.Connecting {
		wcs.SendNotify(NotifyWarning, "Reconnecting to the Hub... stand by")
	} else if stat.ConnectionStatus == hubclient.ConnectFailed {
		// this happens after a server restart as it invalidates user sessions
		// redirect to the login page
		wcs.SendNotify(NotifyWarning, "Connection with Hub refused")
	} else if stat.ConnectionStatus == hubclient.Disconnected {
		wcs.SendNotify(NotifyWarning, "Disconnected")
	} else {
		// catchall
		wcs.SendNotify(NotifyWarning, "Connection failed: "+stat.LastError.Error())
	}
	// connectStatus triggers a reload of the connection status icon.
	// If the connection is lost then the router in HiveovService will redirect to login instead.
	wcs.SendSSE("connectStatus", string(stat.ConnectionStatus))
}

// onMessage notifies SSE clients of incoming messages from the Hub
func (wcs *WebClientSession) onMessage(msg *hubclient.ThingMessage) {
	wcs.mux.RLock()
	defer wcs.mux.RUnlock()

	slog.Info("received message",
		slog.String("type", msg.MessageType),
		slog.String("thingID", msg.ThingID),
		slog.String("name", msg.Name),
		slog.Any("data", msg.Data),
		slog.String("messageID", msg.MessageID))
	if msg.MessageType == vocab.MessageTypeTD {
		// Publish sse event indicating the Thing TD has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this TD:
		//    hx-trigger="sse:{{.Thing.ThingID}}"
		thingAddr := msg.ThingID
		wcs.SendSSE(thingAddr, "")
	} else if msg.MessageType == vocab.MessageTypeProperty {
		// Publish a sse event for each of the properties
		// The UI that displays this event can use this as a trigger to load the
		// property value:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{k}}"
		props := make(map[string]string)
		err := utils.DecodeAsObject(msg.Data, &props)
		if err == nil {
			for k, v := range props {
				thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, k)
				wcs.SendSSE(thingAddr, v)
				thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, k)
				wcs.SendSSE(thingAddr, msg.GetUpdated())
			}
		}
	} else if msg.MessageType == vocab.MessageTypeDeliveryUpdate {
		// report unhandled delivery updates
		// for now just pass it to the notification toaster
		stat := hubclient.DeliveryStatus{}
		_ = utils.DecodeAsObject(msg.Data, &stat)

		// TODO: figure out a way to replace the existing notification if the messageID
		//  is the same (status changes from applied to delivered)
		if stat.Error != "" {
			wcs.SendNotify(NotifyError, stat.Error)
		} else if stat.Progress == vocab.ProgressStatusCompleted {
			wcs.SendNotify(NotifySuccess, "Action successful")
		} else {
			wcs.SendNotify(NotifyWarning, "Action delivery: "+stat.Progress)
		}
	} else {
		// Publish sse event indicating the event affordance or value has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this event:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{$k}}"
		// where $k is the event ID
		eventName := fmt.Sprintf("%s/%s", msg.ThingID, msg.Name)
		wcs.SendSSE(eventName, msg.DataAsText())
		eventName = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Name)
		wcs.SendSSE(eventName, msg.GetUpdated())
	}
}

// RemoveSSEClient removes a disconnected client from the session
// If the session state has changed then store the session data
func (wcs *WebClientSession) RemoveSSEClient(c chan SSEEvent) {
	wcs.mux.Lock()
	defer wcs.mux.Unlock()
	for i, sseClient := range wcs.sseClients {
		if sseClient == c {
			// delete(wcs.sseClients,i)
			wcs.sseClients = append(wcs.sseClients[:i], wcs.sseClients[i+1:]...)
			break
		}
	}
	if wcs.clientState.Changed() {
		err := wcs.SaveState()
		if err != nil {
			wcs.lastError = err
		}

	}
}

// ReplaceHubClient replaces this session's hub client
func (wcs *WebClientSession) ReplaceHubClient(newHC hubclient.IConsumerClient) {
	// ensure the old client is disconnected
	if wcs.hc != nil {
		wcs.hc.Disconnect()
		wcs.hc.SetMessageHandler(nil)
		wcs.hc.SetConnectHandler(nil)
	}
	wcs.hc = newHC
	wcs.hc.SetConnectHandler(wcs.onConnectChange)
	wcs.cts = consumedthing.NewConsumedThingsSession(wcs.hc)
}

// SaveState stores the current client session model using the state service,
// if 'clientModelChanged' is set.
//
// This returns an error if the state service is not reachable.
func (wcs *WebClientSession) SaveState() error {
	if !wcs.clientState.Changed() {
		return nil
	}

	stateCl := stateclient.NewStateClient(wcs.GetHubClient())
	err := stateCl.Set(HiveOViewDataKey, &wcs.clientState)
	if err != nil {
		wcs.lastError = err
		return err
	}
	wcs.clientState.SetChanged(false)
	return err
}

// SetClientData update the hiveoview data model of this client
//func (cs *WebClientSession) SetClientData(data ClientDataModel) {
//	cs.clientState = data
//	cs.clientModelChanged = true
//}

// SendNotify sends a 'notify' event for showing in a toast popup.
// To send an SSE event use SendSSE()
//
//	ntype is the toast notification type: "info", "error", "warning"
func (wcs *WebClientSession) SendNotify(ntype NotifyType, text string) {
	wcs.mux.RLock()
	defer wcs.mux.RUnlock()
	for _, c := range wcs.sseClients {
		c <- SSEEvent{Event: "notify", Payload: string(ntype) + ":" + text}
	}
}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to notify the browser of changes.
func (wcs *WebClientSession) SendSSE(event string, content string) {
	wcs.mux.RLock()
	defer wcs.mux.RUnlock()
	slog.Debug("sending sse event", "event", event, "nr clients", len(wcs.sseClients))
	for _, c := range wcs.sseClients {
		c <- SSEEvent{event, content}
	}
}

// WriteError handles reporting of an error in this session
//
// This logs the error, sends a SSE notification.
// If err is nil then write the httpCode result
// If no httpCode is given then this renders an error message
// If an httpCode is given then this returns the status code
func (wcs *WebClientSession) WriteError(w http.ResponseWriter, err error, httpCode int) {
	if err == nil {
		w.WriteHeader(httpCode)
		return
	}
	slog.Error(err.Error())
	wcs.SendNotify(NotifyError, err.Error())

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
func (wcs *WebClientSession) WritePage(w http.ResponseWriter, buff *bytes.Buffer, err error) {
	if err != nil {
		wcs.WriteError(w, err, http.StatusInternalServerError)
	} else {
		_, _ = buff.WriteTo(w)
	}
}

// NewClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, hc hubclient.IConsumerClient, remoteAddr string) *WebClientSession {
	cs := WebClientSession{
		sessionID:    sessionID,
		remoteAddr:   remoteAddr,
		hc:           hc,
		sseClients:   make([]chan SSEEvent, 0),
		lastActivity: time.Now(),
		clientState:  NewClientDataModel(),
		viewModel:    NewClientViewModel(hc),
		cts:          consumedthing.NewConsumedThingsSession(hc),
	}
	//hc.SetMessageHandler(cs.onMessage)
	hc.SetConnectHandler(cs.onConnectChange)
	// this is a bit quirky but its a transition period
	cs.cts.SetEventHandler(cs.onMessage)

	// restore the session data model
	err := cs.LoadState()
	if err != nil {
		slog.Warn("unable to load client state from state service",
			"clientID", cs.hc.ClientID(), "err", err.Error())
		cs.SendNotify(NotifyWarning, "Unable to restore session: "+err.Error())
		cs.lastError = err
	}

	// TODO: selectively subscribe instead of everything
	err = hc.Subscribe("", "")
	if err != nil {
		cs.lastError = err
	}

	return &cs
}
