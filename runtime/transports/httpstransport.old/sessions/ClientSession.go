package sessions

import (
	"encoding/json"
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

// ClientSession of a client connected over http.
// If this client subscribes using sse then sse event channels are kept.
type ClientSession struct {
	// ID of this session
	sessionID string

	// ClientID is the login ID of the user
	clientID string
	// RemoteAddr of the user
	remoteAddr string
	// track last used time to auto-close inactive sessions
	lastActivity time.Time

	// session mutex for updating sse and activity
	mux sync.RWMutex

	// The SSE event channel for this session.
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent

	// Map of current subscriptions: {thingID}.{name}
	// where name can be a wildcard '+'
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

// CloseSSEChan closes a previously created SSE channel and removes it.
func (cs *ClientSession) CloseSSEChan(c chan SSEEvent) {
	slog.Debug("DeleteSSEChan channel", "clientID", cs.clientID)
	cs.mux.Lock()
	defer cs.mux.Unlock()
	for i, sseClient := range cs.sseClients {
		if sseClient == c {
			cs.sseClients = append(cs.sseClients[:i], cs.sseClients[i+1:]...)
			close(c)
			break
		}
	}
}

// CreateSSEChan creates a new SSE channel to communicate with.
// The channel has a buffer of 1 to allow sending a ping message on connect.
// Call CloseSSEClient to close and clean up
func (cs *ClientSession) CreateSSEChan() chan SSEEvent {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	sseChan := make(chan SSEEvent, 1)
	cs.sseClients = append(cs.sseClients, sseChan)
	return sseChan
}

func (cs *ClientSession) GetClientID() string {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	return cs.clientID
}

// GetNrConnections returns the number of SSE connections for the session
func (cs *ClientSession) GetNrConnections() int {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	return len(cs.sseClients)
}
func (cs *ClientSession) GetSessionID() string {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	return cs.sessionID
}

// IsSubscribed returns true  if this client session has subscribed to events from the Thing and key
func (cs *ClientSession) IsSubscribed(dThingID string, key string) bool {
	cs.mux.RLock()
	defer cs.mux.RUnlock()

	subKey := fmt.Sprintf("%s.%s", dThingID, key)
	_, hasSubscription := cs.subscriptions[subKey]
	if hasSubscription {
		return true
	}
	// check if subscription for this thing with any key exists
	subKey = fmt.Sprintf("%s.+", dThingID)
	_, hasSubscription = cs.subscriptions[subKey]
	if hasSubscription {
		return true
	}
	// check if subscription for any thing with this key exists
	subKey = fmt.Sprintf("+.%s", key)
	_, hasSubscription = cs.subscriptions[subKey]
	if hasSubscription {
		return true
	}
	// check if subscription for any thing with any key exists
	subKey = fmt.Sprintf("+.+")
	_, hasSubscription = cs.subscriptions[subKey]
	if hasSubscription {
		return true
	}
	return hasSubscription
}

// onConnectChange is invoked on disconnect/reconnect
func (cs *ClientSession) onConnectChange(stat hubclient.TransportStatus) {
	slog.Info("connection change",
		slog.String("clientID", stat.ClientID),
		slog.String("status", string(stat.ConnectionStatus)))
	if stat.ConnectionStatus == hubclient.Connected {
		cs.SendSSE("success", "notify", "success:Connection with Hub successful")
	} else if stat.ConnectionStatus == hubclient.Connecting {
		cs.SendSSE("reconnecting", "notify", "warning:Attempt to reconnect to the Hub")
	} else {
		cs.SendSSE("changed", "notify", "warning:Connection changed: "+string(stat.ConnectionStatus))
	}
}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to send events to clients over sse.
// This returns the number of events being sent, or 0 if no client sessions exist
//
// note: when the receiver expects json encoded data, make sure data is of type
// string, not a binary array.
func (cs *ClientSession) SendSSE(messageID string, eventType string, data any) int {
	count := 0
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Debug("hub sending message to client over sse:",
		slog.String("messageID", messageID),
		slog.String("destination clientID", cs.clientID),
		slog.String("eventType", eventType),
		slog.Int("nr connections", len(cs.sseClients)))

	payload, _ := json.Marshal(data)
	for _, c := range cs.sseClients {
		c <- SSEEvent{ // FIXME: put a timeout on it to recover from lockup
			ID:        messageID,
			EventType: eventType,
			Payload:   string(payload),
		}
		count++
	}
	return count
}

// ObserveProperty adds the property observeevent subscription for this session client
//
//	dThingID is the digitwin thingID whose property to observe to, or "" for any
//	name is the property name to observe or "" for all
func (cs *ClientSession) ObserveProperty(dThingID string, name string) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	cs.subscriptions[subKey] = name
}

// SubscribeEvent adds the event subscription for this session client
//
//	dThingID is the digitwin thingID whose events to subscribe to, or "" for any
//	name is the event name to subscribe to or "" for any
func (cs *ClientSession) SubscribeEvent(dThingID string, name string) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	cs.subscriptions[subKey] = name
}

// UnsubscribeEvent removes the event subscription for this session client
// This must match the dThingID and name of SubscribeEvent
func (cs *ClientSession) UnsubscribeEvent(dThingID string, name string) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	delete(cs.subscriptions, subKey)
}

// UnobserveProperty removes the property observe subscription for this session client
// This must match the dThingID and name of ObserveProperty
func (cs *ClientSession) UnobserveProperty(dThingID string, name string) {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	delete(cs.subscriptions, subKey)
}

// UpdateLastActivity sets the current time
func (cs *ClientSession) UpdateLastActivity() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	cs.lastActivity = time.Now()
}

// NewClientSession creates a new client session
// Intended for use by the session manager.
// This subscribes to events for configured agents.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, clientID string, remoteAddr string) *ClientSession {
	cs := ClientSession{
		sessionID:     sessionID,
		clientID:      clientID,
		remoteAddr:    remoteAddr,
		sseClients:    make([]chan SSEEvent, 0),
		lastActivity:  time.Now(),
		subscriptions: make(map[string]string),
	}

	return &cs
}
