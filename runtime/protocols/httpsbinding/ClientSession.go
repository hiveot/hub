package httpsbinding

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
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

// ClientSession of a client connected over http.
// If this client subscribes using sse then sse event channels are kept.
type ClientSession struct {
	// ID of this session
	sessionID string

	// ClientID is the login ID of the user
	clientID string
	// RemoteAddr of the user
	remoteAddr string

	lastActivity time.Time

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
}

// Close the session
// This closes the SSE data channels
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	for _, sseChan := range cs.sseClients {
		close(sseChan)
	}
	cs.sseClients = nil
}

func (cs *ClientSession) GetSessionID() string {
	return cs.sessionID
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
func (cs *ClientSession) onEvent(msg *things.ThingMessage) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Info("received event", slog.String("thingID", msg.ThingID),
		slog.String("id", msg.Key))
	if msg.Key == vocab.EventTypeTD {
		// Publish sse event indicating the Thing TD has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this TD:
		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}"
		_ = cs.SendSSE(msg.ThingID, "")
	} else if msg.Key == vocab.EventTypeProperties {
		// Publish an sse event for each of the properties
		// The UI that displays this event can use this as a trigger to load the
		// property value:
		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}/{{k}}"
		props := make(map[string]string)
		err := json.Unmarshal([]byte(msg.Data), &props)
		if err == nil {
			for k, v := range props {
				thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, k)
				_ = cs.SendSSE(thingAddr, v)
				thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, k)
				_ = cs.SendSSE(thingAddr, msg.GetUpdated())
			}
		}
	} else {
		// Publish sse event indicating the event affordance or value has changed.
		// The UI that displays this event can use this as a trigger to reload the
		// fragment that displays this event:
		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}/{{$k}}"
		// where $k is the event ID
		thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, msg.Key)
		_ = cs.SendSSE(thingAddr, string(msg.Data))
		// TODO: improve on this crude way to update the 'updated' field
		// Can the value contain an object with a value and updated field instead?
		thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Key)
		_ = cs.SendSSE(thingAddr, msg.GetUpdated())
	}
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

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to send events to clients over sse.
func (cs *ClientSession) SendSSE(key string, payload string) error {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Info("sending sse event", "key", key, "nr clients", len(cs.sseClients))
	for _, c := range cs.sseClients {
		c <- SSEEvent{key, payload}
	}
	return nil
}

// NewClientSession creates a new client session
// Intended for use by the session manager.
// This subscribes to events for configured agents.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, clientID string, remoteAddr string) *ClientSession {
	cs := ClientSession{
		sessionID:  sessionID,
		clientID:   clientID,
		remoteAddr: remoteAddr,
		// TODO: assess need for buffering
		sseClients:   make([]chan SSEEvent, 0),
		lastActivity: time.Now(),
	}

	return &cs
}
