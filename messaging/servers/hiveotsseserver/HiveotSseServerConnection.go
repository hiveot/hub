package hiveotsseserver

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
)

type SSEEvent struct {
	EventType string // type of message, e.g. event, action or other
	Payload   string // message content
}

// SSEPingEvent can be used by the server to ping the client that the connection is ready
const SSEPingEvent = "sse-ping"

// HiveotSseServerConnection handles the SSE connection by remote client
//
// The Sse-sc protocol binding uses a 'hiveot' message envelope for sending messages
// between server and consumer.
//
// This implements the IServerConnection interface for sending messages to
// the client over SSE.
type HiveotSseServerConnection struct {
	// Connection information such as clientID, cid, address, protocol etc
	cinfo messaging.ConnectionInfo

	//// connection ID (from header, without clientID prefix)
	//connectionID string
	//
	//// clientID is the account ID of the agent or consumer
	//clientID string

	// connection remote address
	remoteAddr string

	// incoming sse request
	httpReq *http.Request

	isConnected atomic.Bool

	// track last used time to auto-close inactive cm
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// notify client of a connect or disconnect
	connectionHandler messaging.ConnectionHandler
	// handler for requests send by clients
	appRequestHandlerPtr atomic.Pointer[messaging.RequestHandler]
	// handler for responses sent by agents
	responseHandlerPtr atomic.Pointer[messaging.ResponseHandler]
	// handler for notifications sent by agents
	notificationHandlerPtr atomic.Pointer[messaging.NotificationHandler]

	sseChan chan SSEEvent

	subscriptions connections.Subscriptions
	observations  connections.Subscriptions
	//
	correlData map[string]chan any

	// fallback to the SSE mode instead of the SSE-SC mode
	// enabled when subscription headers are present on connect.
	sseFallback bool
}

//type HttpActionStatus struct {
//	CorrelationID string `json:"request_id"`
//	ThingID   string `json:"thingID"`
//	Name      string `json:"name"`
//	Data      any    `json:"data"`
//	Error     string `json:"error"`
//}

// _send sends a request, response or notification message to the client over SSE.
// This is different from the WoT SSE subprotocol in that the payload is the
// message envelope and can carry any operation.
func (c *HiveotSseServerConnection) _send(msgType string, msg any) (err error) {

	payloadJSON, _ := jsoniter.MarshalToString(msg)
	sseMsg := SSEEvent{
		EventType: msgType,
		Payload:   payloadJSON,
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.isConnected.Load() {
		slog.Debug("_send",
			slog.String("to", c.cinfo.ClientID),
			slog.String("MessageType", msgType),
		)
		c.sseChan <- sseMsg
	}
	// as long as the channel exists, delivery will take place
	return nil
}

// Disconnect closes the connection and ends the read loop
func (c *HiveotSseServerConnection) Disconnect() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.isConnected.Load() {
		if c.sseChan != nil {
			close(c.sseChan)
		}
		c.isConnected.Store(false)
	}
}

// GetConnectionInfo returns the client's connection details
func (c *HiveotSseServerConnection) GetConnectionInfo() messaging.ConnectionInfo {
	return c.cinfo
}

// GetConnectURL returns the connection URL of this connection
//func (c *HiveotSseServerConnection) GetConnectURL() string {
//	return c.httpReq.URL.String()
//}

// IsConnected returns the connection status
func (c *HiveotSseServerConnection) IsConnected() bool {
	return c.isConnected.Load()
}

// Handle incoming request messages.
//
// A response is only expected if the request is handled, otherwise nil is returned
// and a response is received asynchronously.
// In case of invoke-action, the response is always an ActionStatus object.
//
// This returns one of 3 options:
// 1. on completion, return handled=true, an optional output
// 2. on error, return handled=true, output optional error details and error the error message
// 3. on async status, return handled=false, output optional, error nil
func (c *HiveotSseServerConnection) onRequestMessage(
	req *messaging.RequestMessage) (handled bool, output *messaging.ResponseMessage, err error) {

	// handle subscriptions
	handled = true
	switch req.Operation {
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		c.subscriptions.Subscribe(req.ThingID, req.Name, req.CorrelationID)
	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
		c.subscriptions.Unsubscribe(req.ThingID, req.Name)
	case wot.OpObserveProperty, wot.OpObserveAllProperties:
		c.observations.Subscribe(req.ThingID, req.Name, req.CorrelationID)
	case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		c.observations.Unsubscribe(req.ThingID, req.Name)
	default:
		handled = false
	}
	if handled {
		// subscription requests dont have output
		err = nil
		return handled, nil, nil
	}
	// not a subscription request, pass it to the application
	hPtr := c.appRequestHandlerPtr.Load()
	if hPtr == nil {
		// internal error
		err = fmt.Errorf("HiveotSseServerConnection:onRequestMessage: no request handler registered")
		return true, nil, err
	}

	resp := (*hPtr)(req, c)
	// responses are optional
	if resp != nil {
		// the hiveot binding returns the response envelope
		output = resp
		handled = true
		err = nil
		if resp.Error != "" {
			err = errors.New(resp.Error)
			handled = true
		}
	} else {
		// no response yet, return a 201
		handled = false
		err = nil
		output = nil
	}
	return handled, output, err
}

// SendNotification sends a notification to the client if subscribed.
func (c *HiveotSseServerConnection) SendNotification(
	notif *messaging.NotificationMessage) (err error) {

	if notif.Operation == wot.OpSubscribeEvent || notif.Operation == wot.OpSubscribeAllEvents {
		correlationID := c.subscriptions.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("SendNotification (subscribed event)",
				slog.String("clientID", c.cinfo.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
			err = c._send(messaging.MessageTypeNotification, notif)
		}
	} else if notif.Operation == wot.OpObserveProperty || notif.Operation == wot.OpObserveAllProperties {
		correlationID := c.observations.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("SendNotification (observed property)",
				slog.String("clientID", c.cinfo.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
			err = c._send(messaging.MessageTypeNotification, notif)
		}
	} else if notif.Operation == wot.OpInvokeAction {
		// action progress update
		slog.Info("SendNotification (action status)",
			slog.String("clientID", c.cinfo.ClientID),
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
		)
		err = c._send(messaging.MessageTypeNotification, notif)
	} else {
		slog.Warn("Unknown notification: " + notif.Operation)
		//err = c._send(msg)
	}
	return err
}

// SendRequest sends a request message to an agent over SSE.
func (c *HiveotSseServerConnection) SendRequest(req *messaging.RequestMessage) error {
	// This simply sends the message as-is
	return c._send(messaging.MessageTypeRequest, req)
}

// SendResponse send a response from server to client over SSE.
func (c *HiveotSseServerConnection) SendResponse(resp *messaging.ResponseMessage) error {
	// This simply sends the message as-is
	return c._send(messaging.MessageTypeResponse, resp)
}

// Serve serves SSE cm.
// This listens for outgoing requests on the given channel
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (c *HiveotSseServerConnection) Serve(w http.ResponseWriter, r *http.Request) {
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
		slog.String("clientID", c.cinfo.ClientID),
		slog.String("connectionID", c.cinfo.ConnectionID),
		slog.String("protocol", r.Proto),
		slog.String("remoteAddr", c.remoteAddr),
	)
	sendLoop := true

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

	// read the message channel for sending messages until it closes
	for sendLoop { // sseMsg := range sseChan {
		select {
		// keep reading to prevent blocking on channel on write
		case sseMsg, ok := <-c.sseChan: // received event
			var err error

			if !ok { // channel was closed by session
				// avoid further writes
				sendLoop = false
				// ending the read loop and returning will close the connection
				break
			}
			slog.Debug("SseConnection: sending sse event to client",
				//slog.String("sessionID", c.sessionID),
				slog.String("clientID", c.cinfo.ClientID),
				slog.String("connectionID", c.cinfo.ConnectionID),
				slog.String("sse eventType", sseMsg.EventType),
			)
			var n int
			n, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.Payload)
			//_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
			//	sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			if err != nil {
				// the connection might be closing.
				// don't exit the loop until the receive-channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("SseConnection: Error writing SSE event",
					slog.String("Event", sseMsg.EventType),
					slog.String("SenderID", c.cinfo.ClientID),
					slog.Int("size", len(sseMsg.Payload)),
				)
			} else {
				slog.Debug("SseConnection: SSE write to client",
					slog.String("SenderID", c.cinfo.ClientID),
					slog.String("Event", sseMsg.EventType),
					slog.Int("N bytes", n))
			}
			w.(http.Flusher).Flush()
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Debug("SseConnection.Serve: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", c.cinfo.ClientID),
		slog.String("connectionID", c.cinfo.ConnectionID),
	)
}

// SetConnectHandler set the connection changed callback. Used by the connection manager.
func (c *HiveotSseServerConnection) SetConnectHandler(cb messaging.ConnectionHandler) {
	c.mux.Lock()
	c.connectionHandler = cb
	c.mux.Unlock()
}

// SetNotificationHandler sets the handler for incoming notification messages from the
// http connection.
//
// Handlers of notifications must register a callback using SetNotificationHandler on the connection.
//
// Note on how this works: The global http server receives notifications as http requests.
// To make it look like the notification came from this connection it looks up the
// connection using the clientID and connectionID and passes the message to the
// registered notification handler on this connection.
//
// By default, the server registers itself as the notification handler when the connection
// is created. It is safe to set a different handler for applications that
// handle each connection separately, for example a server side consumer instance.
func (c *HiveotSseServerConnection) SetNotificationHandler(cb messaging.NotificationHandler) {
	if cb == nil {
		c.notificationHandlerPtr.Store(nil)
	} else {
		c.notificationHandlerPtr.Store(&cb)
	}
}

// SetRequestHandler sets the handler for incoming request messages from the
// http connection.
//
// The hiveot server design requires that the messages are coming from the connections.
// Handlers of requests must register a callback using SetRequestHandler on the connection.
//
// Note on how this works: The global http server receives http requests. To make
// it look like the request came from this connection it looks up the connection
// using the clientID and connectionID and passes the message to the registered
// request handler on this connection.
//
// By default, the server registers itself as the request handler when the connection
// is created. It is safe to set a different request handler for applications that
// handle each connection separately, for example an 'Agent' instance.
func (c *HiveotSseServerConnection) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		c.appRequestHandlerPtr.Store(nil)
	} else {
		c.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler sets the handler for incoming response messages from the
// http connection.
//
// The hiveot server design requires that the messages are coming from the connections.
// Handlers of responses must register a callback using SetResponseHandler on the connection.
//
// Note on how this works: The global http server receives responses as http requests.
// To make it look like the response came from this connection it looks up the
// connection using the clientID and connectionID and passes the message to the
// registered response handler on this connection.
//
// By default, the server registers itself as the response handler when the connection
// is created. It is safe to set a different response handler for applications that
// handle each connection separately, for example a server side consumer instance.
func (c *HiveotSseServerConnection) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		c.responseHandlerPtr.Store(nil)
	} else {
		c.responseHandlerPtr.Store(&cb)
	}
}

// NewHiveotSseConnection creates a new SSE connection instance.
// This implements the IServerConnection interface.
func NewHiveotSseConnection(clientID string, cid string, remoteAddr string,
	httpReq *http.Request, sseFallback bool) *HiveotSseServerConnection {

	cinfo := messaging.ConnectionInfo{
		CaCert:       nil,
		ClientID:     clientID,
		ConnectionID: cid,
		ConnectURL:   httpReq.URL.String(),
		//ProtocolType: messaging.ProtocolTypeHiveotSSE,
		Timeout: 0,
	}
	c := &HiveotSseServerConnection{
		cinfo:         cinfo,
		remoteAddr:    remoteAddr,
		httpReq:       httpReq,
		lastActivity:  time.Now(),
		mux:           sync.RWMutex{},
		observations:  connections.Subscriptions{},
		subscriptions: connections.Subscriptions{},
		correlData:    make(map[string]chan any),
		sseFallback:   sseFallback,
	}
	c.isConnected.Store(true)

	// interface check
	var _ messaging.IServerConnection = c
	return c
}
