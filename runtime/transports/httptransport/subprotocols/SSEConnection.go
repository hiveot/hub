package subprotocols

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/digitwin"
	"golang.org/x/exp/slices"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type SSEEvent struct {
	EventType string // type of message, eg event, action or other
	ID        string // event ID
	Payload   string // message content
}

// SSEConnection of an authenticated SSE connection over http.
// This is used for both the sse-sc and sse bindings.
//
// This implements the IProtocolConnection interface
type SSEConnection struct {
	// connection ID of this session
	connectionID string

	// connection belongs to this session
	sessionID string

	// ClientID is the login ID of the agent or consumer
	clientID string

	// RemoteAddr of the user
	//remoteAddr string
	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	sseChan chan SSEEvent

	// eventSubscriptions of this connection in the form {dThingID}.{name}
	// This uses "+" as wildcards
	eventSubscriptions []string

	// propSubscriptions of this connection in the form {dThingID}.{name}
	// This uses "+" as wildcards
	propSubscriptions []string
}

// _publish sends the property or event update to the connected consumer
func (c *SSEConnection) _publish(messageType string,
	dThingID, name string, data any, messageID string) {

	var payload []byte = nil
	if data != nil {
		payload, _ = json.Marshal(data)
	}
	topic := fmt.Sprintf("%s/%s/%s", dThingID, name, messageID)
	msg := SSEEvent{
		EventType: messageType,
		ID:        topic,
		Payload:   string(payload),
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.sseChan != nil {
		c.sseChan <- msg
	}
}

// _send sends the action or write request for the thing to the agent
func (c *SSEConnection) _send(messageType string,
	agentID, thingID, name string, data any, messageID string) (status string, err error) {

	var payload []byte = nil
	if data != nil {
		payload, _ = json.Marshal(data)
	}
	topic := fmt.Sprintf("%s/%s/%s/%s", agentID, thingID, name, messageID)
	msg := SSEEvent{
		EventType: messageType,
		ID:        topic,
		Payload:   string(payload),
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.sseChan != nil {
		c.sseChan <- msg
	}
	return digitwin.StatusPending, nil
}

// Close closes the connection and ends the read loop
func (c *SSEConnection) Close() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.sseChan != nil {
		close(c.sseChan)
	}
}

// GetClientID returns the client's connection ID
func (c *SSEConnection) GetClientID() string {
	return c.clientID
}

// InvokeAction sends the action request for the thing to the agent
func (c *SSEConnection) InvokeAction(
	agentID, thingID, name string, data any, messageID string) (
	status string, output any, err error) {

	status, err = c._send(vocab.MessageTypeAction, agentID, thingID, name, data, messageID)
	return status, nil, err
}

// IsSubscribed returns true if subscription for thing and name exists
// If dThingID or name are empty, then "+" is used as wildcard
func (c *SSEConnection) IsSubscribed(subs []string, dThingID string, name string) bool {
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := dThingID + "." + name
	for _, v := range subs {
		if v == subKey {
			return true
		}
	}
	return false
}

// PublishEvent send an event to subscribers
func (c *SSEConnection) PublishEvent(dThingID, name string, data any, messageID string) {
	if c.IsSubscribed(c.eventSubscriptions, dThingID, name) {
		c._publish(vocab.MessageTypeEvent, dThingID, name, data, messageID)
	}
}

// PublishProperty send a property change update to subscribers
func (c *SSEConnection) PublishProperty(dThingID, name string, data any, messageID string) {
	if c.IsSubscribed(c.propSubscriptions, dThingID, name) {
		c._publish(vocab.MessageTypeProperty, dThingID, name, data, messageID)
	}
}

// ObserveProperty adds a subscription for a thing property
func (c *SSEConnection) ObserveProperty(dThingID string, name string) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	c.propSubscriptions = append(c.propSubscriptions, subKey)
}

// Serve serves SSE connections.
// This listens for outgoing requests on the given channel
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (c *SSEConnection) Serve(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	// establish a client event channel for sending messages back to the client
	c.sseChan = make(chan SSEEvent, 1)

	// Send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := SSEEvent{EventType: hubclient.PingMessage}
	c.sseChan <- pingEvent

	slog.Info("SseConnection. New SSE connection",
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("clientID", c.clientID),
		slog.String("protocol", r.Proto),
		slog.String("sessionID", c.sessionID),
	)
	//var sseMsg SSEEvent

	readLoop := true

	// close the channel when the connection drops
	go func() {
		select {
		case <-r.Context().Done(): // remote client connection closed
			slog.Debug("SseConnection: Remote client disconnected (read context)")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			c.Close()
		}
	}()

	// read the message channel until it closes
	for readLoop { // sseMsg := range sseChan {
		select {
		// keep reading to prevent blocking on channel on write
		case sseMsg, ok := <-c.sseChan: // received event
			var err error

			if !ok { // channel was closed by session
				// avoid further writes
				readLoop = false
				// ending the read loop and returning will close the connection
				break
			}
			slog.Debug("SseConnection: sending sse event to client",
				slog.String("sessionID", c.sessionID),
				slog.String("clientID", c.clientID),
				slog.String("sse eventType", sseMsg.EventType),
			)
			// write the message with or without messageID
			if sseMsg.ID == "" {
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
					sseMsg.EventType, sseMsg.Payload)
			} else {
				_, err = fmt.Fprintf(w, "event: %s\nid:%s\ndata: %s\n\n",
					sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			}
			if err != nil {
				// the connection might be closing.
				// don't exit the loop until the receive channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("Error writing SSE event", "ID", sseMsg.ID,
					"size", len(sseMsg.Payload))
			}
			w.(http.Flusher).Flush()
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Info("SseConnection: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", c.clientID),
	)
}

// SubscribeEvent handles a subscription request for an event
func (c *SSEConnection) SubscribeEvent(dThingID string, name string) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	c.eventSubscriptions = append(c.eventSubscriptions, subKey)
}

// UnsubscribeEvent removes an event subscription
// dThingID and name must match those of ObserveProperty
func (c *SSEConnection) UnsubscribeEvent(dThingID string, name string) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	i := slices.Index(c.eventSubscriptions, subKey)
	if i >= 0 {
		c.eventSubscriptions = slices.Delete(c.eventSubscriptions, i, i)
	}
}

// UnobserveProperty removes a property subscription
// dThingID and name must match those of ObserveProperty
func (c *SSEConnection) UnobserveProperty(dThingID string, name string) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}

	subKey := fmt.Sprintf("%s.%s", dThingID, name)
	i := slices.Index(c.propSubscriptions, subKey)
	if i >= 0 {
		c.propSubscriptions = slices.Delete(c.propSubscriptions, i, i)
	}
}

// WriteProperty sends the property change request to the agent
func (c *SSEConnection) WriteProperty(
	agentID, thingID, name string, data any, messageID string) (status string, err error) {

	status, err = c._send(vocab.MessageTypeProperty, agentID, thingID, name, data, messageID)
	return status, err
}

func NewSSEConnection(connectionID, sessionID, clientID string) *SSEConnection {
	c := &SSEConnection{
		connectionID:       connectionID,
		sessionID:          sessionID,
		clientID:           clientID,
		lastActivity:       time.Time{},
		mux:                sync.RWMutex{},
		eventSubscriptions: make([]string, 0),
		propSubscriptions:  make([]string, 0),
	}
	return c
}
