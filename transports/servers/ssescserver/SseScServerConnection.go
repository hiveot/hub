package ssescserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type SSEEvent struct {
	EventType string // type of message, eg event, action or other
	Payload   string // message content
}

// SSEPingEvent can be used by the server to ping the client that the connection is ready
const SSEPingEvent = "sseping"

// SseScServerConnection handles the SSE connection by remote client
//
// The Sse-sc protocol binding uses a 'hiveot' message envelope for sending messages
// between server and consumer.
//
// This implements the IServerConnection interface for sending messages to
// the client over SSE.
type SseScServerConnection struct {
	// connection ID (from header, without clientID prefix)
	connectionID string

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
	observations  connections.Subscriptions
	//
	correlData map[string]chan any

	// fallback to the SSE mode instead of the SSE-SC mode
	// enabled when subscription headers are present on connect.
	sseFallback bool
}

//type HttpActionStatus struct {
//	RequestID string `json:"request_id"`
//	ThingID   string `json:"thingID"`
//	Name      string `json:"name"`
//	Data      any    `json:"data"`
//	Error     string `json:"error"`
//}

// _send sends a request, response or notification message to the client over SSE.
// This is different from the WoT SSE subprotocol in that the payload is the
// message envelope and can carry any operation.
func (c *SseScServerConnection) _send(msgType string, msg any) (err error) {

	var payload []byte = nil
	payload, _ = jsoniter.Marshal(msg)
	sseMsg := SSEEvent{
		EventType: msgType,
		Payload:   string(payload),
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		slog.Debug("_send",
			slog.String("to", c.clientID),
			slog.String("MessageType", msgType),
		)
		c.sseChan <- sseMsg
	}
	// as long as the channel exists, delivery will take place
	return nil
}

// Disconnect closes the connection and ends the read loop
func (c *SseScServerConnection) Disconnect() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		close(c.sseChan)
		c.isClosed.Store(true)
	}
}

// GetClientID returns the client's account ID
func (c *SseScServerConnection) GetClientID() string {
	return c.clientID
}

// GetConnectionID returns the clients connection ID unique within the sessions
func (c *SseScServerConnection) GetConnectionID() string {
	return c.connectionID
}

// GetProtocolType returns the protocol used in this connection
func (c *SseScServerConnection) GetProtocolType() string {
	return transports.ProtocolTypeSSESC
}

// GetSessionID returns the client's authentication session ID
//func (c *SseScServerConnection) GetSessionID() string {
//	return c.sessionID
//}

//// IsSubscribed returns true if subscription for thing and name exists
//// If dThingID or name are empty, then "+" is used as wildcard
//func (c *SseScServerConnection) IsSubscribed(subs []string, dThingID string, name string) bool {
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
func (c *SseScServerConnection) ObserveProperty(dThingID string, name string) {
	c.observations.Subscribe(dThingID, name)
}

// SendNotification sends a notification message if the client is subscribed
func (c *SseScServerConnection) SendNotification(noti transports.NotificationMessage) {

	switch noti.Operation {
	case wot.HTOpUpdateTD:
		// update the TD if the client is subscribed to its events
		if c.subscriptions.IsSubscribed(noti.ThingID, "") {
			_ = c._send(transports.MessageTypeNotification, noti)
		}
	case wot.HTOpEvent:
		if c.subscriptions.IsSubscribed(noti.ThingID, noti.Name) {
			_ = c._send(transports.MessageTypeNotification, noti)
		}
	case wot.HTOpUpdateProperty, wot.HTOpUpdateMultipleProperties:
		if c.observations.IsSubscribed(noti.ThingID, noti.Name) {
			_ = c._send(transports.MessageTypeNotification, noti)
		}
	default:
		slog.Error("SendNotification: Unknown notification operation",
			"op", noti.Operation,
			"thingID", noti.ThingID,
			"to", c.clientID)
	}
}

// SendRequest sends a request message to an agent over SSE
func (c *SseScServerConnection) SendRequest(req transports.RequestMessage) error {
	// This simply sends the message as-is
	return c._send(transports.MessageTypeRequest, req)
}

// SendResponse send a response (action status) from server to client.
func (c *SseScServerConnection) SendResponse(resp transports.ResponseMessage) error {
	// This simply sends the message as-is
	return c._send(transports.MessageTypeResponse, resp)
}

// Serve serves SSE connections.
// This listens for outgoing requests on the given channel
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (c *SseScServerConnection) Serve(w http.ResponseWriter, r *http.Request) {
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

	// _send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := SSEEvent{EventType: SSEPingEvent}
	c.mux.Lock()
	c.sseChan <- pingEvent
	c.mux.Unlock()

	slog.Debug("SseConnection.Serve new SSE connection",
		slog.String("clientID", c.clientID),
		slog.String("connectionID", c.connectionID),
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
			c.Disconnect()

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
				slog.String("connectionID", c.connectionID),
				slog.String("sse eventType", sseMsg.EventType),
			)
			var n int
			n, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.Payload)
			//_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
			//	sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			if err != nil {
				// the connection might be closing.
				// don't exit the loop until the receive channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("SseConnection: Error writing SSE event",
					slog.String("Event", sseMsg.EventType),
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
		slog.String("connectionID", c.connectionID),
	)
}

// SubscribeEvent handles a subscription request for an event
func (c *SseScServerConnection) SubscribeEvent(dThingID string, name string) {
	c.subscriptions.Subscribe(dThingID, name)
}

// UnsubscribeEvent removes an event subscription
// dThingID and name must match those of ObserveProperty
func (c *SseScServerConnection) UnsubscribeEvent(dThingID string, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}

// UnobserveProperty removes a property subscription
// dThingID and name must match those of ObserveProperty
func (c *SseScServerConnection) UnobserveProperty(dThingID string, name string) {
	c.observations.Unsubscribe(dThingID, name)
}

// WriteProperty sends the property change request to the agent
//func (c *SseScServerConnection) WriteProperty(
//	thingID, name string, data any, requestID string, senderID string) (status string, err error) {
//
//	status, err = c._send(vocab.OpWriteProperty, thingID, name, data, requestID, senderID)
//	return status, err
//}

// NewSSEConnection creates a new SSE connection instance.
// This implements the IServerConnection interface.
func NewSSEConnection(clientID string, cid string, remoteAddr string, sseFallback bool) *SseScServerConnection {
	connectionID := clientID + "-" + cid // -> must match subscribe/observe requests

	c := &SseScServerConnection{
		connectionID:  connectionID,
		clientID:      clientID,
		remoteAddr:    remoteAddr,
		lastActivity:  time.Time{},
		mux:           sync.RWMutex{},
		observations:  connections.Subscriptions{},
		subscriptions: connections.Subscriptions{},
		correlData:    make(map[string]chan any),
		sseFallback:   sseFallback,
	}
	// interface check
	var _ transports.IServerConnection = c
	return c
}
