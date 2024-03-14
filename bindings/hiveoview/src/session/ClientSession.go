package session

import (
	"fmt"
	"github.com/hiveot/hub/core/state/stateclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"sync"
	"time"
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
	hc *hubclient.HubClient
	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channels for this session
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent
}

func (cs *ClientSession) AddSSEClient(c chan SSEEvent) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	cs.sseClients = append(cs.sseClients, c)

	go func() {
		if cs.IsActive() {
			cs.SendSSE("notify", "success:Connected to the Hub")
		} else {
			cs.SendSSE("notify", "error:Not connected to the Hub")
		}
	}()
}

// Close the session and save its state.
// This closes the hub connection and SSE data channels
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	for _, sseChan := range cs.sseClients {
		close(sseChan)
	}
	cs.hc.Disconnect()
	cs.sseClients = nil
}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (cs *ClientSession) GetStatus() transports.HubTransportStatus {
	status := cs.hc.GetStatus()
	return status
}

// GetHubClient returns the hub client connection for use in pub/sub
func (cs *ClientSession) GetHubClient() *hubclient.HubClient {
	return cs.hc
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
func (cs *ClientSession) IsActive() bool {
	status := cs.hc.GetStatus()
	return status.ConnectionStatus == transports.Connected ||
		status.ConnectionStatus == transports.Connecting
}

// onConnectChange is invoked on disconnect/reconnect
func (cs *ClientSession) onConnectChange(stat transports.HubTransportStatus) {
	slog.Info("connection change",
		slog.String("clientID", stat.ClientID),
		slog.String("status", string(stat.ConnectionStatus)))
	if stat.ConnectionStatus == transports.Connected {
		cs.SendSSE("notify", "success:Connection with Hub successful")
	} else if stat.ConnectionStatus == transports.Connecting {
		cs.SendSSE("notify", "warning:Attempt to reconnect to the Hub")
	} else {
		cs.SendSSE("notify", "warning:Connection changed: "+string(stat.ConnectionStatus))
	}
}

// onEvent passes incoming events from the Hub to the SSE client(s)
func (cs *ClientSession) onEvent(msg *things.ThingValue) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	// FIXME: HOW TO IMPLEMENT DATA BINDING WITH HTMX fragments?
	// A: use alpine.js databinding. Include JS objects that the element binds to
	//    and use sse events to trigger a reload of the object.
	//
	//    pro: * one and two-way data binding provided by Alpine
	//    con: * risk duplicating server/client state
	//         * dependent on the Alpine-js kitchen sink
	//
	// B: use sse event to reload data associated fragments.
	//    (one event can affect multiple fragments)
	//
	//    pro: * isolation between data updates and fragment reload (separation of concerns)
	//         * support 1-many relationship for data-fragments
	//    con: * have to manually manage many fragments and event names
	//         * all fragments must have unique IDs
	//         * fragment reloads can cause unintended side effects like layout changes.
	//           for example the open/close state of a 'details' element is reset after reload.
	//
	// Q: is there a need for client side state to bind to?
	// Q: does htmx has an extension to facilitate data binding?
	//		hx-trigger="customers from:body" ??? how to trigger specific TD ID?
	//      ? hx-trigger="sse:<thingID>" could this work?

	if msg.Name == transports.EventNameTD {
		// TODO: send the current view a TD changed event
		// FIXME: can the event have spaces? => NO!
		thingAddr := fmt.Sprintf("%s/%s", msg.AgentID, msg.ThingID)
		_ = cs.SendSSE(thingAddr, "")
	} else if msg.Name == transports.EventNameProps {
		// TODO: send the current view a properties changed event, if applicable
		// TODO: how are the other views that display the value updated?
	} else {
		// value changed event
		// TODO: send the current view a value changed event, if applicable
		// TODO: how are the other views that display the value updated?
	}
	slog.Info("received event", "id", msg.Name)
	// TODO: determine how events are consumed
	// SSE's usually expect an HTML snippet, not json data
	//_ = cs.SendSSE(msg.Name, string(msg.Data))
}

func (cs *ClientSession) RemoveSSEClient(c chan SSEEvent) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	for i, sseClient := range cs.sseClients {
		if sseClient == c {
			// delete(cs.sseClients,i)
			cs.sseClients = append(cs.sseClients[:i], cs.sseClients[i+1:]...)
			break
		}
	}
}

// ReplaceHubClient replaces this session's hub client
func (cs *ClientSession) ReplaceHubClient(newHC *hubclient.HubClient) {
	// ensure the old client is disconnected
	if cs.hc != nil {
		cs.hc.Disconnect()
		cs.hc.SetEventHandler(nil)
		cs.hc.SetConnectionHandler(nil)
	}
	cs.hc = newHC
	cs.hc.SetConnectionHandler(cs.onConnectChange)
	cs.hc.SetEventHandler(cs.onEvent)
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

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to notify the browser of changes.
func (cs *ClientSession) SendSSE(event string, content string) error {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Info("sending sse event", "event", event, "nr clients", len(cs.sseClients))
	for _, c := range cs.sseClients {
		c <- SSEEvent{event, content}
	}
	return nil
}

// NewClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
// This subscribes to events for configured agents.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, hc *hubclient.HubClient, remoteAddr string) *ClientSession {
	cs := ClientSession{
		sessionID:  sessionID,
		clientID:   hc.ClientID(),
		remoteAddr: remoteAddr,
		hc:         hc,
		// TODO: assess need for buffering
		sseClients:   make([]chan SSEEvent, 0),
		lastActivity: time.Now(),
	}
	hc.SetEventHandler(cs.onEvent)
	hc.SetConnectionHandler(cs.onConnectChange)

	// restore the session data model
	stateCl := stateclient.NewStateClient(hc)
	found, err := stateCl.Get(hc.ClientID(), &cs.clientModel)
	_ = found
	_ = err
	if len(cs.clientModel.Agents) > 0 {
		for _, agent := range cs.clientModel.Agents {
			// subscribe to TD and value events
			err = hc.SubEvents(agent, "", "")
		}
	} else {
		// no agent set so subscribe to all agents
		err = hc.SubEvents("", "", "")
	}
	// subscribe to configured agents
	return &cs
}
