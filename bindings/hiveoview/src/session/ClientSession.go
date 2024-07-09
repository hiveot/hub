package session

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/state/stateclient"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type NotifyType string

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

// ClientSession of a web client containing a hub connection
type ClientSession struct {
	// ID of this session
	sessionID string

	// Client subscription and dashboard model, loaded from the state service
	clientModel ClientModel

	// ClientID is the login ID of the user
	clientID string
	// RemoteAddr of the user
	remoteAddr string

	lastActivity time.Time

	// The associated hub client for pub/sub
	hc hubclient.IHubClient
	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channels for this session
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent
}

func (cs *ClientSession) AddSSEClient(c chan SSEEvent) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	cs.sseClients = append(cs.sseClients, c)

	go func() {
		if cs.IsActive() {
			cs.SendNotify(NotifySuccess, "Connected to the Hub")
		} else {
			cs.SendNotify(NotifyError, "Not connected to the Hub")
		}
	}()
}

// Close the session and save its state.
// This closes the hub connection and SSE data channels
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	for _, sseChan := range cs.sseClients {
		close(sseChan)
	}
	cs.sseClients = nil
	cs.mux.Unlock()
	cs.hc.Disconnect()
}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (cs *ClientSession) GetStatus() hubclient.TransportStatus {
	status := cs.hc.GetStatus()
	return status
}

// GetHubClient returns the hub client connection for use in pub/sub
func (cs *ClientSession) GetHubClient() hubclient.IHubClient {
	return cs.hc
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
func (cs *ClientSession) IsActive() bool {
	status := cs.hc.GetStatus()
	return status.ConnectionStatus == hubclient.Connected ||
		status.ConnectionStatus == hubclient.Connecting
}

// onConnectChange is invoked on disconnect/reconnect
func (cs *ClientSession) onConnectChange(stat hubclient.TransportStatus) {
	lastErrText := ""
	if stat.LastError != nil {
		lastErrText = stat.LastError.Error()
	}
	slog.Info("onConnectChange",
		slog.String("clientID", stat.ClientID),
		slog.String("status", string(stat.ConnectionStatus)),
		slog.String("lastError", lastErrText))

	if stat.ConnectionStatus == hubclient.Connected {
		cs.SendNotify(NotifySuccess, "Connection established with the Hub")
	} else if stat.ConnectionStatus == hubclient.Connecting {
		cs.SendNotify(NotifyWarning, "Reconnecting to the Hub... stand by")
	} else if stat.ConnectionStatus == hubclient.ConnectFailed {
		// this happens after a server restart as it invalidates user sessions
		// redirect to the login page
		cs.SendNotify(NotifyWarning, "Connection with Hub refused")
	} else if stat.ConnectionStatus == hubclient.Disconnected {
		cs.SendNotify(NotifyWarning, "Disconnected")
	} else {
		// catchall
		cs.SendNotify(NotifyWarning, "Connection failed: "+stat.LastError.Error())
	}
	// connectStatus triggers a reload of the connection status icon.
	// If the connection is lost then the router in HiveovService will redirect to login instead.
	cs.SendSSE("connectStatus", string(stat.ConnectionStatus))
}

// onMessage passes incoming messages from the Hub to the SSE client(s)
func (cs *ClientSession) onMessage(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()

	slog.Info("received message",
		slog.String("type", msg.MessageType),
		slog.String("thingID", msg.ThingID),
		slog.String("key", msg.Key),
		//slog.Any("data", msg.Data),
		slog.String("messageID", msg.MessageID))
	if msg.Key == vocab.EventTypeTD {
		// Publish sse event indicating the Thing TD has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this TD:
		//    hx-trigger="sse:{{.Thing.ThingID}}"
		thingAddr := msg.ThingID
		cs.SendSSE(thingAddr, "")
	} else if msg.Key == vocab.EventTypeProperties {
		// Publish an sse event for each of the properties
		// The UI that displays this event can use this as a trigger to load the
		// property value:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{k}}"
		props := make(map[string]string)
		err := msg.Decode(&props)
		if err == nil {
			for k, v := range props {
				thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, k)
				cs.SendSSE(thingAddr, v)
				thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, k)
				cs.SendSSE(thingAddr, msg.GetUpdated())
			}
		}
	} else if msg.Key == vocab.EventTypeDeliveryUpdate {
		// report unhandled delivery updates
		// for now just pass it to the notification toaster
		stat := hubclient.DeliveryStatus{}
		_ = msg.Decode(&stat)
		// TODO: figure out a way to replace the existing notification if the messageID
		//  is the same (status changes from applied to delivered)
		if stat.Error != "" {
			cs.SendNotify(NotifyError, stat.Error)
		} else if stat.Progress == hubclient.DeliveryCompleted {
			cs.SendNotify(NotifySuccess, "Action successful")
		} else {
			cs.SendNotify(NotifyWarning, "Action delivery: "+stat.Progress)
		}
	} else {
		// Publish sse event indicating the event affordance or value has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this event:
		//    hx-trigger="sse:{{.Thing.ThingID}}/{{$k}}"
		// where $k is the event ID
		thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, msg.Key)
		cs.SendSSE(thingAddr, msg.DataAsText())
		// TODO: improve on this crude way to update the 'updated' field
		// Can the value contain an object with a value and updated field instead?
		// htmx sse-swap does allow cherry picking the content unfortunately.
		thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Key)
		cs.SendSSE(thingAddr, msg.GetUpdated())
	}
	return stat.Completed(msg, nil, nil)
}

func (cs *ClientSession) RemoveSSEClient(c chan SSEEvent) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	for i, sseClient := range cs.sseClients {
		if sseClient == c {
			// delete(cs.sseClients,i)
			cs.sseClients = append(cs.sseClients[:i], cs.sseClients[i+1:]...)
			break
		}
	}
}

// ReadTD is a simple helper to read and unmarshal a TD
func (cs *ClientSession) ReadTD(thingID string) (*things.TD, error) {
	td := &things.TD{}
	tdJson, err := digitwin.DirectoryReadTD(cs.hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	return td, err
}

// ReplaceHubClient replaces this session's hub client
func (cs *ClientSession) ReplaceHubClient(newHC hubclient.IHubClient) {
	// ensure the old client is disconnected
	if cs.hc != nil {
		cs.hc.Disconnect()
		cs.hc.SetMessageHandler(nil)
		cs.hc.SetConnectHandler(nil)
	}
	cs.hc = newHC
	cs.hc.SetConnectHandler(cs.onConnectChange)
	cs.hc.SetMessageHandler(cs.onMessage)
}

// SaveState stores the current model to the server
func (cs *ClientSession) SaveState() error {
	stateCl := stateclient.NewStateClient(cs.GetHubClient())
	err := stateCl.Set(cs.clientID, &cs.clientModel)
	//if err != nil {
	//	slog.Error("unable to save session state",
	//		slog.String("clientID", cs.clientID),
	//		slog.String("err", err.Error()))
	//}
	return err
}

func (cs *ClientSession) SendNotify(ntype NotifyType, text string) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	for _, c := range cs.sseClients {
		c <- SSEEvent{Event: "notify", Payload: string(ntype) + ":" + text}
	}
}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to notify the browser of changes.
func (cs *ClientSession) SendSSE(event string, content string) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Debug("sending sse event", "event", event, "nr clients", len(cs.sseClients))
	for _, c := range cs.sseClients {
		c <- SSEEvent{event, content}
	}
}

// WriteError handles reporting of an error in this session
// This logs the error, sends a SSE notification.
// If no httpCode is given then this renders an error message
// If an httpCode is given then this returns the status code
func (cs *ClientSession) WriteError(w http.ResponseWriter, err error, httpCode int) {
	slog.Error(err.Error())
	cs.SendNotify(NotifyError, err.Error())
	if httpCode == 0 {
		output := "Oops: " + err.Error()
		// Write returns http OK
		_, _ = w.Write([]byte(output))
		return
	}

	http.Error(w, err.Error(), httpCode)
}

// NewClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
// This subscribes to events for configured agents.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, hc hubclient.IHubClient, remoteAddr string) *ClientSession {
	cs := ClientSession{
		sessionID:  sessionID,
		clientID:   hc.ClientID(),
		remoteAddr: remoteAddr,
		hc:         hc,
		// TODO: assess need for buffering
		sseClients:   make([]chan SSEEvent, 0),
		lastActivity: time.Now(),
	}
	hc.SetMessageHandler(cs.onMessage)
	hc.SetConnectHandler(cs.onConnectChange)

	// restore the session data model
	stateCl := stateclient.NewStateClient(hc)
	found, err := stateCl.Get(hc.ClientID(), &cs.clientModel)
	_ = found
	_ = err
	if len(cs.clientModel.Agents) > 0 {
		// TODO: with digitwin it is no longer possible to subscribe to an agent, its all or nothing
		//
		//for _, agent := range cs.clientModel.Agents {
		// subscribe to TD and value events
		err = hc.Subscribe("", "")
		//}
	} else {
		// no agent set so subscribe to all agents
		err = hc.Subscribe("", "")
	}
	// subscribe to configured agents
	return &cs
}
