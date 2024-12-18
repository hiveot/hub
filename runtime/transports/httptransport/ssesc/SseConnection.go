package ssesc

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/connections"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type SSEEvent struct {
	EventType string // type of message, eg event, action or other
	ID        string // message topic: {thingID}/{name}/{requestID}
	Payload   string // message content
}

// SSEConnection of an authenticated SSE connection over http.
// This is used for both the sse-sc and sse bindings.
//
// This implements the IClientConnection interface
type SSEConnection struct {
	// connection ID (from header, without clientID prefix)
	clcid string

	// clientID is the account ID of the agent or consumer
	clientID string

	// connection remote address
	remoteAddr string

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	sseChan  chan SSEEvent
	isClosed atomic.Bool

	subscriptions connections.Subscriptions
}

// _send sends the action or write request for the thing to the agent
// The SSE event type is: messageType, where
// The event ID is {thingID}/{name}/{requestID}/{senderID}
func (c *SSEConnection) _send(operation string, thingID, name string,
	data any, requestID string, senderID string) (status string, err error) {

	var payload []byte = nil
	if data != nil {
		payload, _ = json.Marshal(data)
	}
	eventID := fmt.Sprintf("%s/%s/%s/%s", thingID, name, senderID, requestID)
	msg := SSEEvent{
		EventType: operation,
		ID:        eventID,
		Payload:   string(payload),
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		slog.Debug("_send",
			slog.String("to", c.clientID),
			slog.String("MessageType", operation),
			slog.String("eventID", eventID),
		)
		c.sseChan <- msg
	}
	// as long as the channel exists, delivery will take place
	// FIXME: guarantee delivery
	// todo: detect race conditions; or accept the small risk of delivery to a closing connection?
	return vocab.RequestDelivered, nil
}

// Close closes the connection and ends the read loop
func (c *SSEConnection) Close() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		close(c.sseChan)
		c.isClosed.Store(true)
	}
}

// GetClientID returns the client's account ID
func (c *SSEConnection) GetClientID() string {
	return c.clientID
}

// GetCLCID returns the clients connection ID unique within the sessions
func (c *SSEConnection) GetConnectionID() string {
	return c.clcid
}

// GetSessionID returns the client's authentication session ID
//func (c *SSEConnection) GetSessionID() string {
//	return c.sessionID
//}

// InvokeAction sends the action request for the thing to the agent over SSE
func (c *SSEConnection) InvokeAction(
	thingID, name string, data any, requestID string, senderID string) (
	status string, output any, err error) {

	status, err = c._send(vocab.OpInvokeAction, thingID, name, data, requestID, senderID)
	return status, nil, err
}

//// IsSubscribed returns true if subscription for thing and name exists
//// If dThingID or name are empty, then "+" is used as wildcard
//func (c *SSEConnection) IsSubscribed(subs []string, dThingID string, name string) bool {
//	if dThingID == "" {
//		dThingID = "+"
//	}
//	if name == "" {
//		name = "+"
//	}
//	subKey := dThingID + "." + name
//	for _, v := range subs {
//		if v == subKey {
//			return true
//		}
//	}
//	return false
//}

// ObserveProperty adds a subscription for a thing property
func (c *SSEConnection) ObserveProperty(dThingID string, name string) {
	c.subscriptions.Observe(dThingID, name)
}

// PublishActionStatus sends an action progress update to the client
// If an error is provided this sends the error, otherwise the output value
func (c *SSEConnection) PublishActionStatus(stat hubclient.RequestStatus, agentID string) error {
	if stat.CorrelationID == "" {
		slog.Error("PublishActionStatus without requestID", "agentID", agentID)
	}
	_, err := c._send(vocab.HTOpUpdateActionStatus, stat.ThingID, stat.Name,
		stat, stat.CorrelationID, agentID)
	return err
}

// PublishEvent send an event to subscribers
func (c *SSEConnection) PublishEvent(
	dThingID, name string, data any, requestID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		_, _ = c._send(vocab.HTOpPublishEvent, dThingID, name, data, requestID, agentID)
	}
}

// PublishProperty send a property change update to observers
// if name is empty then data contains a map of property key-value pairs
func (c *SSEConnection) PublishProperty(
	dThingID, name string, data any, requestID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		_, _ = c._send(vocab.HTOpUpdateProperty, dThingID, name, data, requestID, agentID)
	}
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
	w.Header().Set("Content-Encoding", "none") //https://stackoverflow.com/questions/76375157/client-not-receiving-server-sent-events-from-express-js-server

	// establish a client event channel for sending messages back to the client
	c.mux.Lock()
	c.sseChan = make(chan SSEEvent, 1)
	c.mux.Unlock()

	// Send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := SSEEvent{EventType: hubclient.PingMessage, ID: "pingID"}
	c.mux.Lock()
	c.sseChan <- pingEvent
	c.mux.Unlock()

	slog.Debug("SseConnection.Serve new SSE connection",
		slog.String("clientID", c.clientID),
		slog.String("clcid", c.clcid),
		slog.String("protocol", r.Proto),
		slog.String("remoteAddr", c.remoteAddr),
	)
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
				//slog.String("sessionID", c.sessionID),
				slog.String("clientID", c.clientID),
				slog.String("clcid", c.clcid),
				slog.String("sseMsg", sseMsg.ID),
				slog.String("sse eventType", sseMsg.EventType),
			)
			// write the message with or without requestID
			if sseMsg.ID == "" {
				// force a requestID to avoid go-sse client injecting the last eventID,
				// which can mess things up.
				sseMsg.ID = "-"
			}
			var n int
			n, err = fmt.Fprintf(w, "event: %s\nid:%s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			//_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
			//	sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			if err != nil {
				// the connection might be closing.
				// don't exit the loop until the receive channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("SseConnection: Error writing SSE event",
					slog.String("Event", sseMsg.EventType),
					slog.String("SSE ID", sseMsg.ID),
					slog.String("SenderID", c.clientID),
					slog.Int("size", len(sseMsg.Payload)),
				)
			} else {
				slog.Debug("SseConnection: SSE write to client",
					slog.String("SenderID", c.clientID),
					slog.String("Event", sseMsg.EventType),
					slog.Int("N bytes", n))
			}
			w.(http.Flusher).Flush()
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Debug("SseConnection.Serve: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", c.clientID),
		slog.String("clcid", c.clcid),
	)
}

// SubscribeEvent handles a subscription request for an event
func (c *SSEConnection) SubscribeEvent(dThingID string, name string) {
	c.subscriptions.Subscribe(dThingID, name)
}

// UnsubscribeEvent removes an event subscription
// dThingID and name must match those of ObserveProperty
func (c *SSEConnection) UnsubscribeEvent(dThingID string, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}

// UnobserveProperty removes a property subscription
// dThingID and name must match those of ObserveProperty
func (c *SSEConnection) UnobserveProperty(dThingID string, name string) {
	c.subscriptions.Unobserve(dThingID, name)
}

// WriteProperty sends the property change request to the agent
func (c *SSEConnection) WriteProperty(
	thingID, name string, data any, requestID string, senderID string) (status string, err error) {

	status, err = c._send(vocab.OpWriteProperty, thingID, name, data, requestID, senderID)
	return status, err
}

// NewSSEConnection creates a new SSE connection instance.
// This implements the IClientConnection interface.
func NewSSEConnection(clientID string, cid string, remoteAddr string) *SSEConnection {
	clcid := clientID + "-" + cid // -> must match subscribe/observe requests

	c := &SSEConnection{
		clcid:         clcid,
		clientID:      clientID,
		remoteAddr:    remoteAddr,
		lastActivity:  time.Time{},
		mux:           sync.RWMutex{},
		subscriptions: connections.Subscriptions{},
	}
	return c
}
