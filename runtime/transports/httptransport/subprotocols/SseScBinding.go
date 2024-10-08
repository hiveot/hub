package subprotocols

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/runtime/transports/httptransport/sessions"
	"log/slog"
	"net/http"
	"sync"
)

// SSE-SC subprotocol binding
type SseScBinding struct {
	sm *sessions.SessionManager

	// session mutex for updating connections
	mux sync.RWMutex

	// map of sse connections by connection id
	connections map[string]IClientConnection
}

// determine the connection ID and return the associated connection
func (b *SseScBinding) getConnection(r *http.Request) (IClientConnection, error) {
	sessID, clientID, err := sessions.GetSessionIdFromContext(r)
	_ = clientID
	if err != nil {
		return nil, err
	}
	// FIXME: use connectionID to allow multiple connections per session
	c, found := b.connections[sessID]
	if !found {
		return nil, fmt.Errorf("no connection for session '%s'", sessID)
	}
	return c, nil
}

// HandleConnect handles a new sse-sc connection
func (b *SseScBinding) HandleConnect(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login.
	sessID, clientID, err := sessions.GetSessionIdFromContext(r)

	if err != nil {
		slog.Warn("No session available yet, telling client to delay retry to 10 seconds")

		// set retry to a large number
		// see https://javascript.info/server-sent-events#reconnection
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"10000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		//w.Write([]byte(errMsg))
		w.(http.Flusher).Flush()
		return
	}
	// for now use the sessionID as the connectionID.
	// this wont work for multiple connections though. Need to add something unique
	// that is matched in the subscription requests.
	connID := sessID

	// create a new sse session
	c := NewSSEConnection(connID, sessID, clientID)
	b.mux.Lock()
	b.connections[connID] = c
	b.mux.Unlock()
	c.Serve(w, r)
	delete(b.connections, connID)
	// don't return until the connection is closed
}

// HandleObserveProperty handles a property observe request for one or all properties
func (b *SseScBinding) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	clientID, dThingID, name, _, err := GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleObserve", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleObserve",
		slog.String("clientID", clientID),
		slog.String("thingID", dThingID),
		slog.String("name", name))
	//slog.String("sessionID", cs.GetSessionID()))
	//slog.Int("nr sse connections", cs.GetNrConnections()))
	//cs.ObserveProperty(thingID, name)

	c, err := b.getConnection(r)
	if err == nil {
		c.ObserveProperty(dThingID, name)
	}
}

// HandleObserveAllProperties adds a property subscription
func (b *SseScBinding) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	b.HandleObserveProperty(w, r)
}

// HandleSubscribeEvent handles a subscription request for one or all events
func (b *SseScBinding) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	clientID, dThingID, name, _, err := GetRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscribe", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleSubscribe",
		slog.String("clientID", clientID),
		slog.String("thingID", dThingID),
		slog.String("name", name))

	c, err := b.getConnection(r)
	if err == nil {
		c.SubscribeEvent(dThingID, name)
	}
}

// HandleSubscribeAllEvents adds a subscription to all events
func (b *SseScBinding) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	b.HandleSubscribeEvent(w, r)
}

// HandleUnobserveAllProperties handles removal of all property observe subscriptions
func (b *SseScBinding) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	b.HandleUnobserveProperty(w, r)
}

// HandleUnobserveProperty handles removal of one property observe subscriptions
func (b *SseScBinding) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnobserveProperty")
	_, dThingID, name, _, err := GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c, err := b.getConnection(r)
	if err == nil {
		c.UnobserveProperty(dThingID, name)
	}
}

// HandleUnsubscribeAllEvents removes the subscription
func (b *SseScBinding) HandleUnsubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	b.HandleUnsubscribeEvent(w, r)
}

// HandleUnsubscribeEvent handles removal of one or all event subscriptions
func (b *SseScBinding) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribe")
	_, dThingID, name, _, err := GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c, err := b.getConnection(r)
	if err == nil {
		c.UnsubscribeEvent(dThingID, name)
	}
}

// InvokeAction sends the action request for the thing to the agent
func (b *SseScBinding) InvokeAction(
	agentID, thingID, name string, data any, messageID string) (
	status string, output any, err error) {

	// determine which connection is of the agent
	for _, c := range b.connections {
		if c.GetClientID() == agentID {
			return c.InvokeAction(thingID, name, data, messageID)
		}
	}

	return digitwin.StatusFailed, nil,
		fmt.Errorf("agent '%s' does not have a connection", agentID)
}

// PublishEvent send an event to subscribers
func (b *SseScBinding) PublishEvent(dThingID, name string, data any, messageID string) {
	for _, sseConn := range b.connections {
		sseConn.PublishEvent(dThingID, name, data, messageID)
	}
}

// PublishProperty send a property change update to subscribers
func (b *SseScBinding) PublishProperty(dThingID, name string, data any, messageID string) {
	for _, sseConn := range b.connections {
		sseConn.PublishProperty(dThingID, name, data, messageID)
	}
}
func (b *SseScBinding) SendActionResult(clientID string, stat hubclient.DeliveryStatus) (err error) {

	// determine which connection is of the consumer
	for _, sseConn := range b.connections {
		if sseConn.GetClientID() == clientID {
			return sseConn.SendActionResult(stat)
		}
	}
	return fmt.Errorf("not implemented")
}

// WriteProperty sends the write request for the thing property to the agent
func (b *SseScBinding) WriteProperty(
	agentID, thingID, name string, data any, messageID string) (
	status string, err error) {

	// determine which connection is of the agent
	for _, sseConn := range b.connections {
		if sseConn.GetClientID() == agentID {
			return sseConn.WriteProperty(thingID, name, data, messageID)
		}
	}

	return digitwin.StatusFailed,
		fmt.Errorf("agent '%s' does not have a connection", agentID)
}

// NewSseScBinding returns a new SSE-SC sub-protocol binding
func NewSseScBinding(sm *sessions.SessionManager) *SseScBinding {
	b := &SseScBinding{
		sm:          sm,
		connections: make(map[string]IClientConnection),
	}
	return b
}
