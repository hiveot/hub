package sessions

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"sync"
	"time"
)

type SSEEvent struct {
	EventType string // type of message, eg event, action or other
	ID        string // event ID
	Payload   string // message content
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
	sequenceNr   int

	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channels for this session
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent

	// ThingID's subscribed to
	subscriptions map[string]string
}

// Close the session
// This closes the SSE data channels
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	slog.Info("Closing client session", "clientID", cs.clientID)
	for _, sseChan := range cs.sseClients {
		close(sseChan)
	}
	cs.sseClients = nil
}

// CreateSSEChan creates a new SSE channel to communicate with.
// Call DeleteSSEClient to close and clean up
func (cs *ClientSession) CreateSSEChan() chan SSEEvent {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	sseChan := make(chan SSEEvent)
	cs.sseClients = append(cs.sseClients, sseChan)
	return sseChan
}

// DeleteSSEChan deletes a previously created SSE channel and closes it.
func (cs *ClientSession) DeleteSSEChan(c chan SSEEvent) {
	slog.Debug("DeleteSSEChan channel", "clientID", cs.clientID)
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

func (cs *ClientSession) GetClientID() string {
	return cs.clientID
}

// GetNrConnections returns the number of SSE connections for the session
func (cs *ClientSession) GetNrConnections() int {
	return len(cs.sseClients)
}
func (cs *ClientSession) GetSessionID() string {
	return cs.sessionID
}

// IsSubscribed returns true  if this client session has subscribed to events from the Thing and optionally key
// TODO: add more fine-grained multi-key subscription
func (cs *ClientSession) IsSubscribed(dThingID string, key string) bool {
	subKey, hasSubscription := cs.subscriptions[dThingID]
	if !hasSubscription {
		// also check for wildcard subscription
		subKey, hasSubscription = cs.subscriptions["+"]
		if !hasSubscription {
			return false
		}
	}
	// subscribed to any event
	if subKey == "" || subKey == "+" || key == "" {
		return true
	}
	return subKey == key
}

// onConnectChange is invoked on disconnect/reconnect
func (cs *ClientSession) onConnectChange(stat hubclient.TransportStatus) {
	slog.Info("connection change",
		slog.String("clientID", stat.ClientID),
		slog.String("status", string(stat.ConnectionStatus)))
	if stat.ConnectionStatus == hubclient.Connected {
		cs.SendSSE("notify", "success:Connection with Hub successful")
	} else if stat.ConnectionStatus == hubclient.Connecting {
		cs.SendSSE("notify", "warning:Attempt to reconnect to the Hub")
	} else {
		cs.SendSSE("notify", "warning:Connection changed: "+string(stat.ConnectionStatus))
	}
}

// onEvent passes incoming events from the Hub to the SSE client(s)
//func (cs *ClientSession) onEvent(msg *things.ThingMessage) {
//	cs.mux.RLock()
//	defer cs.mux.RUnlock()
//	slog.Info("received event", slog.String("thingID", msg.ThingID),
//		slog.String("id", msg.Key))
//	if msg.Key == vocab.EventTypeTD {
//		// Publish sse event indicating the Thing TD has changed.
//		// The UI that displays this event can use this as a trigger to reload the
//		// fragment that displays this TD:
//		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}"
//		_ = cs.SendSSE(msg.ThingID, "")
//	} else if msg.Key == vocab.EventTypeProperties {
//		// Publish an sse event for each of the properties
//		// The UI that displays this event can use this as a trigger to load the
//		// property value:
//		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}/{{k}}"
//		props := make(map[string]string)
//		err := json.Unmarshal([]byte(msg.Data), &props)
//		if err == nil {
//			for k, v := range props {
//				thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, k)
//				_ = cs.SendSSE(thingAddr, v)
//				thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, k)
//				_ = cs.SendSSE(thingAddr, msg.GetUpdated())
//			}
//		}
//	} else {
//		// Publish sse event indicating the event affordance or value has changed.
//		// The UI that displays this event can use this as a trigger to reload the
//		// fragment that displays this event:
//		//    hx-trigger="sse:{{.Thing.AgentID}}/{{.Thing.ThingID}}/{{$k}}"
//		// where $k is the event ID
//		thingAddr := fmt.Sprintf("%s/%s", msg.ThingID, msg.Key)
//		_ = cs.SendSSE(thingAddr, string(msg.Data))
//		// TODO: improve on this crude way to update the 'updated' field
//		// Can the value contain an object with a value and updated field instead?
//		thingAddr = fmt.Sprintf("%s/%s/updated", msg.ThingID, msg.Key)
//		_ = cs.SendSSE(thingAddr, msg.GetUpdated())
//	}
//}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to send events to clients over sse.
// This returns the number of events being sent, or 0 if no client sessions exist
func (cs *ClientSession) SendSSE(eventType string, payload string) int {
	count := 0
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Debug("hub sending message to client over sse:",
		slog.String("destination clientID", cs.clientID),
		slog.String("eventType", eventType),
		slog.Int("nr connections", len(cs.sseClients)))
	cs.sequenceNr++
	eventID := fmt.Sprintf("%d", cs.sequenceNr)
	for _, c := range cs.sseClients {
		c <- SSEEvent{
			ID:        eventID,
			EventType: eventType,
			Payload:   payload,
		}
		count++
	}
	return count
}

// Subscribe adds the event subscription for this session client
func (cs *ClientSession) Subscribe(dThingID string, key string) {
	cs.subscriptions[dThingID] = key
}

// Unsubscribe removes the event subscription for this session client
func (cs *ClientSession) Unsubscribe(dThingID string, key string) {
	delete(cs.subscriptions, dThingID)
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
		sseClients:    make([]chan SSEEvent, 0),
		lastActivity:  time.Now(),
		subscriptions: make(map[string]string),
	}

	return &cs
}
