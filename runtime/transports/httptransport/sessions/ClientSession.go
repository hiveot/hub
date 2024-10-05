package sessions

import (
	"log/slog"
	"sync"
	"time"
)

// PushMessage for sending messages to the remote agent or consumer
type PushMessage struct {
	MessageType string // type of message, eg "event", "property", "action" or other

	ThingID string // the thing the message is for or from
	Name    string // name of the event/prop/action affordance

	MessageID string // optional message ID
	Payload   any    // optional message content
}

// ClientSession of an authenticated client connected over http.
type ClientSession struct {
	// ID of this session
	sessionID string

	// ClientID is the login ID of the agent or consumer
	clientID string

	// RemoteAddr of the user
	//remoteAddr string
	// track last used time to auto-close inactive sessions
	lastActivity time.Time

	// session mutex for updating endpoints and activity
	mux sync.RWMutex

	// The connection endpoints of the client.
	// These are handlers of the sub-protocol that implement a return channel
	// from the server to the client. Used by both agents and consumers.
	//endpoints []subprotocols.IProtocolConnection

	// Map of current subscriptions: {thingID}.{name}
	// where name can be a wildcard '+'
	//subscriptions map[string]string
}

// AddEndpoint adds a new endpoint to communicate through.
// The channel has a buffer of 1 to allow sending a ping message on connect.
// Call CloseEndpoint to close and clean up
//func (cs *ClientSession) AddEndpoint(handler subprotocols.IProtocolConnection) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	cs.endpoints = append(cs.endpoints, handler)
//}

// Close the session endpoints
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	slog.Info("Closing client session", "clientID", cs.clientID)
	//for _, endpoint := range cs.endpoints {
	//	cs.RemoveEndpoint(endpoint)
	//	endpoint.Close()
	//}
	//cs.endpoints = make([]subprotocols.IProtocolConnection, 0)
}

func (cs *ClientSession) GetClientID() string {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	return cs.clientID
}

// GetNrConnections returns the number of endpoints for the session
//
//	func (cs *ClientSession) GetNrConnections() int {
//		cs.mux.RLock()
//		defer cs.mux.RUnlock()
//		return len(cs.endpoints)
//	}
func (cs *ClientSession) GetSessionID() string {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	return cs.sessionID
}

// HandleActionFlow invokes an action on the client if connected
// This returns the delivery status and optional output, or an error if delivery is not available.
//func (cs *ClientSession) HandleActionFlow(
//	agentID string, thingID string, name string, input any, messageID string) (
//	status string, output any, err error) {
//
//	cs.mux.RLock()
//	defer cs.mux.RUnlock()
//	if len(cs.endpoints) == 0 {
//		return digitwin.StatusFailed, nil, fmt.Errorf(
//			"HandleActionFlow: Agent '%s' not reachable", agentID)
//	}
//	for _, c := range cs.endpoints {
//		status, output, err = c.HandleActionFlow(agentID, thingID, name, input, messageID)
//		if err != nil {
//			return status, output, err
//		}
//	}
//	// return the last error
//	return status, output, err
//}
//
//// IsSubscribed returns true  if this client session has subscribed to events
//// or observing properties from the Thing and event/property name.
//// TBD: is there a need to separete event subscription and observing properties?
//func (cs *ClientSession) IsSubscribed(dThingID string, name string) bool {
//	cs.mux.RLock()
//	defer cs.mux.RUnlock()
//
//	subKey := fmt.Sprintf("%s.%s", dThingID, name)
//	_, hasSubscription := cs.subscriptions[subKey]
//	if hasSubscription {
//		return true
//	}
//	// check if subscription for this thing with any name exists
//	subKey = fmt.Sprintf("%s.+", dThingID)
//	_, hasSubscription = cs.subscriptions[subKey]
//	if hasSubscription {
//		return true
//	}
//	// check if subscription for any thing with this name exists
//	subKey = fmt.Sprintf("+.%s", name)
//	_, hasSubscription = cs.subscriptions[subKey]
//	if hasSubscription {
//		return true
//	}
//	// check if subscription for any thing with any name exists
//	subKey = fmt.Sprintf("+.+")
//	_, hasSubscription = cs.subscriptions[subKey]
//	if hasSubscription {
//		return true
//	}
//	return hasSubscription
//}

// PublishEvent sends an event message to the connected consumer
//func (cs *ClientSession) PublishEvent(
//	thingID string, name string, value any, messageID string) {
//
//	for _, c := range cs.endpoints {
//		c.PublishEvent(thingID, name, value, messageID)
//	}
//}

// PublishProperty sends a property update to the connected consumer
//func (cs *ClientSession) PublishProperty(
//	thingID string, name string, value any, messageID string) {
//
//	for _, c := range cs.endpoints {
//		c.PublishProperty(thingID, name, value, messageID)
//	}
//}

// ObserveProperty adds the property observeevent subscription for this session client
//
//	dThingID is the digitwin thingID whose property to observe to, or "" for any
//	name is the property name to observe or "" for all
//func (cs *ClientSession) ObserveProperty(dThingID string, name string) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	if name == "" {
//		name = "+"
//	}
//
//	subKey := fmt.Sprintf("%s.%s", dThingID, name)
//	cs.subscriptions[subKey] = name
//}

// RemoveEndpoint removes a previously added endpoint.
//func (cs *ClientSession) RemoveEndpoint(handler subprotocols.IProtocolConnection) {
//	slog.Debug("CloseEndpoint", "clientID", cs.clientID)
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	for i, endpoint := range cs.endpoints {
//		if endpoint == handler {
//			cs.endpoints = append(cs.endpoints[:i], cs.endpoints[i+1:]...)
//			break
//		}
//	}
//}

// SubscribeEvent adds the event subscription for this session client
//
//	dThingID is the digitwin thingID whose events to subscribe to, or "" for any
//	name is the event name to subscribe to or "" for any
//func (cs *ClientSession) SubscribeEvent(dThingID string, name string) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	if name == "" {
//		name = "+"
//	}
//
//	subKey := fmt.Sprintf("%s.%s", dThingID, name)
//	cs.subscriptions[subKey] = name
//}

// UnsubscribeEvent removes the event subscription for this session client
// This must match the dThingID and name of SubscribeEvent
//func (cs *ClientSession) UnsubscribeEvent(dThingID string, name string) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	if name == "" {
//		name = "+"
//	}
//	subKey := fmt.Sprintf("%s.%s", dThingID, name)
//	delete(cs.subscriptions, subKey)
//}

// UnobserveAllProperties removes the observe subscription for all properties
// This must match the dThingID
//func (cs *ClientSession) UnobserveAllProperties(dThingID string) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	subKey := fmt.Sprintf("%s.+", dThingID)
//	delete(cs.subscriptions, subKey)
//}

// UnobserveProperty removes the property observe subscription for this session client
// This must match the dThingID and name of ObserveProperty
//func (cs *ClientSession) UnobserveProperty(dThingID string, name string) {
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	if name == "" {
//		name = "+"
//	}
//	subKey := fmt.Sprintf("%s.%s", dThingID, name)
//	delete(cs.subscriptions, subKey)
//}

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
func NewClientSession(sessionID string, clientID string) *ClientSession {
	cs := ClientSession{
		sessionID: sessionID,
		clientID:  clientID,
		//endpoints:    make([]subprotocols.IProtocolConnection, 0),
		lastActivity: time.Now(),
		//subscriptions: make(map[string]string),
	}

	return &cs
}
