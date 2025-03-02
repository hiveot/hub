package wssserver

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type WSSMessage map[string]any

// WssServerConnection is  the server side instance of a connection by a client.
// This implements the IServerConnection interface for sending messages to
// agent or consumers.
type WssServerConnection struct {
	// Connection information such as clientID, cid, address, protocol etc
	cinfo messaging.ConnectionInfo

	// connection ID
	//connectionID string

	// clientID is the account ID of the agent or consumer
	//clientID string

	// connection request remote address
	httpReq *http.Request

	isConnected atomic.Bool

	// track last used time to auto-close inactive cm
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// handler for requests send by clients
	requestHandlerPtr atomic.Pointer[messaging.RequestHandler]
	// handler for responses sent by agents
	responseHandlerPtr atomic.Pointer[messaging.ResponseHandler]
	// notify client of a connect or disconnect
	connectionHandlerPtr atomic.Pointer[messaging.ConnectionHandler]

	// event subscriptions and property observations by consumers
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions

	// converter for request/response messages
	messageConverter messaging.IMessageConverter

	// underlying websocket connection
	wssConn *websocket.Conn
}

// _send encodes and sends the websocket message to the connected client
func (c *WssServerConnection) _send(msg any) (err error) {

	if !c.isConnected.Load() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", c.cinfo.ClientID)
		slog.Warn(err.Error())
	} else {
		raw, _ := jsoniter.MarshalToString(msg)
		// websockets do not allow concurrent write
		c.mux.Lock()
		defer c.mux.Unlock()
		err = c.wssConn.WriteMessage(websocket.TextMessage, []byte(raw))
		if err != nil {
			err = fmt.Errorf("_send write error: %s", err)
		}
	}
	return err
}

// Disconnect closes the connection and ends the read loop
func (c *WssServerConnection) Disconnect() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.isConnected.Load() {
		c.onConnection(false, nil)
		_ = c.wssConn.Close()
	}
}

// GetConnectionInfo returns the client's connection details
func (c *WssServerConnection) GetConnectionInfo() messaging.ConnectionInfo {
	return c.cinfo
}

// IsConnected returns the connection status
func (c *WssServerConnection) IsConnected() bool {
	return c.isConnected.Load()
}
func (c *WssServerConnection) onConnection(connected bool, err error) {
	c.isConnected.Store(connected)
	chPtr := c.connectionHandlerPtr.Load()
	if chPtr != nil {
		(*chPtr)(connected, err, c)
	}
}

// onMessage handles an incoming websocket message
// The message is converted into a request or response and passed on to the registered handler.
// Messages handled by the transport binding:
// - Ping
// - (Un)ObserveProperty and (Un)ObserveAllProperties
// - (Un)SubscribeEvent and (Un)SubscribeAllEvents
func (c *WssServerConnection) onMessage(raw []byte) {
	var err error
	c.mux.Lock()
	c.lastActivity = time.Now()
	c.mux.Unlock()
	var resp *messaging.ResponseMessage

	// both non-agents and agents send requests
	req := c.messageConverter.DecodeRequest(raw)
	if req != nil {
		// sender is identified by the server, not the client
		// note that this field is still useful for services that need to know the sender
		req.SenderID = c.cinfo.ClientID
		switch req.Operation {
		case wot.HTOpPing:
			resp = req.CreateResponse("pong", nil)

		case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
			c.subscriptions.Subscribe(req.ThingID, req.Name, req.CorrelationID)
			resp = req.CreateResponse(nil, nil)

		case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
			c.subscriptions.Unsubscribe(req.ThingID, req.Name)
			resp = req.CreateResponse(nil, nil)

		case wot.OpObserveProperty, wot.OpObserveAllProperties:
			c.observations.Subscribe(req.ThingID, req.Name, req.CorrelationID)
			resp = req.CreateResponse(nil, nil)

		case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
			c.observations.Unsubscribe(req.ThingID, req.Name)
			resp = req.CreateResponse(nil, nil)
		default:
			rhPtr := c.requestHandlerPtr.Load()
			if rhPtr != nil {
				resp = (*rhPtr)(req, c)
			}
		}
		err = c.SendResponse(resp)
	} else {
		// only agents send responses
		resp = c.messageConverter.DecodeResponse(raw)
		resp.SenderID = c.cinfo.ClientID
		rhPtr := c.responseHandlerPtr.Load()
		if rhPtr != nil {
			err = (*rhPtr)(resp)
		}
	}
	if err != nil {
		slog.Warn("Error handling websocket message", "err", err.Error())
	}
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (c *WssServerConnection) ReadLoop(ctx context.Context, wssConn *websocket.Conn) {

	//var readLoop atomic.Bool
	c.onConnection(true, nil)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WssServerConnection.ReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			c.onConnection(false, nil)
		}
	}()
	// read messages from the client until the connection closes
	for c.isConnected.Load() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			c.onConnection(false, err)
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to free up the socket
		go c.onMessage(raw)
	}
}

// SendRequest sends the request to the client (agent).
//
// Intended to be used on cm that are agents for Things and connect to the hub
// as a client (connection reversal).
// If this server is the Thing agent then there is no need for this method.
//
// If this returns an error then no request was sent.
func (c *WssServerConnection) SendRequest(req *messaging.RequestMessage) error {
	msg, err := c.messageConverter.EncodeRequest(req)
	if err == nil {
		err = c._send(msg)
	}
	return err
}

// SendResponse sends a response to the remote client.
// If this returns an error then no response was sent.
func (c *WssServerConnection) SendResponse(resp *messaging.ResponseMessage) (err error) {

	// FIXME: response from zwavejs ping is not received properly by hiveoview
	// Feb 20 08:17:23.2900 WRN WssClientConnection.go:159 HandleWssMessage: Message is not a request or response
	// server hiveot-wss type (MessageType) field is empty

	slog.Debug("SendResponse (server->client)",
		slog.String("clientID", c.cinfo.ClientID),
		slog.String("correlationID", resp.CorrelationID),
		slog.String("operation", resp.Operation),
		slog.String("name", resp.Name),
		slog.String("status", resp.Status),
		slog.String("type", resp.MessageType),
		slog.String("senderID", resp.SenderID),
	)

	msg, err := c.messageConverter.EncodeResponse(resp)
	if err == nil {
		err = c._send(msg)
	}
	return err
}

// SendNotification sends a response to the client if subscribed.
// this is a response to a long-running subscription request
// If this returns an error then no response was sent.
func (c *WssServerConnection) SendNotification(resp messaging.ResponseMessage) {

	//slog.Info("SendNotification",
	//	slog.String("clientID", c.clientID),
	//	slog.String("correlationID", resp.CorrelationID),
	//	slog.String("operation", resp.Operation),
	//	slog.String("senderID", resp.SenderID),
	//)

	if resp.Operation == wot.OpSubscribeEvent || resp.Operation == wot.OpSubscribeAllEvents {
		correlationID := c.subscriptions.GetSubscription(resp.ThingID, resp.Name)
		if correlationID != "" {
			resp.CorrelationID = correlationID
			_ = c.SendResponse(&resp)
		}
	} else if resp.Operation == wot.OpObserveProperty || resp.Operation == wot.OpObserveAllProperties {
		correlationID := c.observations.GetSubscription(resp.ThingID, resp.Name)
		if correlationID != "" {
			resp.CorrelationID = correlationID
			_ = c.SendResponse(&resp)
		}
	} else {
		slog.Warn("Unknown notification: " + resp.Operation)
	}
	return
}

func (c *WssServerConnection) SetConnectHandler(cb messaging.ConnectionHandler) {
	if cb == nil {
		c.connectionHandlerPtr.Store(nil)
	} else {
		c.connectionHandlerPtr.Store(&cb)
	}
}
func (c *WssServerConnection) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		c.requestHandlerPtr.Store(nil)
	} else {
		c.requestHandlerPtr.Store(&cb)
	}
}
func (c *WssServerConnection) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		c.responseHandlerPtr.Store(nil)
	} else {
		c.responseHandlerPtr.Store(&cb)
	}
}

// NewWSSServerConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IServerConnection interface.
func NewWSSServerConnection(
	clientID string, r *http.Request,
	wssConn *websocket.Conn,
	messageConverter messaging.IMessageConverter,
) *WssServerConnection {

	cid := "WSS" + shortid.MustGenerate()

	cinfo := messaging.ConnectionInfo{
		CaCert:       nil,
		ClientID:     clientID,
		ConnectionID: cid,
		ConnectURL:   r.URL.String(),
		ProtocolType: messageConverter.GetProtocolType(),
		Timeout:      0,
	}
	c := &WssServerConnection{
		wssConn: wssConn,
		cinfo:   cinfo,
		//clientID:         clientID,
		messageConverter: messageConverter,
		httpReq:          r,
		lastActivity:     time.Time{},
		mux:              sync.RWMutex{},
		observations:     connections.Subscriptions{},
		subscriptions:    connections.Subscriptions{},
	}
	return c
}
