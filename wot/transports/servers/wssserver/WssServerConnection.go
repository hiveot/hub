package wssserver

import (
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/wssclient"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type WSSMessage map[string]any

// WssServerConnection is a the server side instance of a connection by a client.
// This implements the IServerConnection interface for sending messages to
// agent or consumers.
type WssServerConnection struct {
	// connection ID
	connectionID string

	// clientID is the account ID of the agent or consumer
	clientID string

	// connection request remote address
	req *http.Request

	// gorilla websocket connection
	wssConn *websocket.Conn

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// handler for passing request to a single destination
	// a reply is expected (asynchronously)
	messageHandler transports.ServerMessageHandler

	isClosed atomic.Bool

	// event subscriptions and property observations by consumers
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions
}

// _error sends a websocket error message to the connected client
//func (c *WssServerConnection) _error(err error, code int, requestID string) {
//	wssMsg := wssclient.ErrorMessage{
//		MessageType: wssclient.MsgTypeError,
//		Title:       err.Error(),
//		RequestID:   requestID,
//		Status:      strconv.Itoa(code),
//	}
//	c._send(wssMsg)
//}

// _send sends the websocket message to the connected client
func (c *WssServerConnection) _send(wssMsg interface{}) {

	if c.isClosed.Load() {
		slog.Error("_send: connection with client is now closed",
			slog.String("to", c.clientID),
		)
	} else {
		slog.Info("_send",
			slog.String("to", c.clientID),
		)

		msgJSON := c.Marshal(wssMsg)
		// websockets do not allow concurrent write
		c.mux.Lock()
		defer c.mux.Unlock()
		err := c.wssConn.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			slog.Error("_send write error", "err", err.Error())
		}
	}
}

// Disconnect closes the connection and ends the read loop
func (c *WssServerConnection) Disconnect() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		c.isClosed.Store(true)
		_ = c.wssConn.Close()
	}
}

// GetConnectionID returns the client's unique connection ID
func (c *WssServerConnection) GetConnectionID() string {
	return c.connectionID
}

// GetClientID returns the client's account ID
func (c *WssServerConnection) GetClientID() string {
	return c.clientID
}

// GetProtocolType returns the type of protocol used in this connection
func (c *WssServerConnection) GetProtocolType() string {
	return transports.ProtocolTypeWSS
}

// SendNotification send an event or property update to subscribers
// This returns an error if the notification could not be delivered to the client.
func (c *WssServerConnection) SendNotification(
	operation string, dThingID, name string, data any) {

	msg, err := wssclient.OpToMessage(operation,
		dThingID, name, nil, data, "")

	if err != nil {
		slog.Error("SendNotification: Unknown operation. Ignored.", "op", operation)
		return
	}

	switch operation {
	case wot.HTOpPublishEvent:
		if c.subscriptions.IsSubscribed(dThingID, name) {
			c._send(msg)
		}
	case wot.HTOpUpdateProperty, wot.HTOpMultipleProperties:
		if c.observations.IsSubscribed(dThingID, name) {
			c._send(msg)
		}
	default:
		slog.Error("SendNotification: Unknown notification operation",
			"op", operation,
			"thingID", dThingID,
			"to", c.clientID)
	}
}

// SendError sends an error response to the client.
func (c *WssServerConnection) SendError(
	thingID, name string, errResponse string, requestID string) {

	if requestID == "" {
		slog.Error("SendError without requestID", "clientID", c.clientID)
	}
	msg := wssclient.ErrorMessage{
		ThingID:     thingID,
		MessageType: wssclient.MsgTypeError,
		Title:       name + " error",
		RequestID:   requestID,
		Detail:      errResponse,
		//Timestamp:   time.Now().Format(wot.RFC3339Milli),
	}
	c._send(msg)
}

// SendRequest sends the request to the client (agent).
// Intended to be used on clients that are agents for Things.
// If this returns an error then no request will was sent.
func (c *WssServerConnection) SendRequest(
	operation string, thingID, name string, input any, requestID string) {
	msg, err := wssclient.OpToMessage(operation, thingID, name, nil, input, requestID)
	if err != nil {
		return
	}
	c._send(msg)
}

// SendResponse sends an action status update to the client.
// If the status is RequestFailed then output is an error, otherwise the output value
// If this returns an error then no request will was sent.
func (c *WssServerConnection) SendResponse(
	thingID, name string, output any, requestID string) {

	if requestID == "" {
		slog.Error("SendResponse to '%s' without requestID",
			"to", c.clientID)
	}
	msg := wssclient.ActionStatusMessage{
		ThingID:     thingID,
		MessageType: wssclient.MsgTypeActionStatus,
		Name:        name,
		RequestID:   requestID,
		Output:      output,
		Timestamp:   time.Now().Format(wot.RFC3339Milli),
	}
	c._send(msg)
}

// PublishProperty publishes a new property value clients that observe it
//func (c *WssServerConnection) PublishProperty(
//	dThingID string, name string, data any, requestID string) {
//
//	if c.observations.IsSubscribed(dThingID, name) {
//		msg := wssbinding.PropertyMessage{
//			ThingID:     dThingID,
//			MessageType: wssbinding.MsgTypePropertyReading,
//			Name:        name,
//			RequestID:   requestID,
//			Data:        data,
//			Timestamp:   time.Now().Format(wot.RFC3339Milli),
//		}
//		_ = c._send(msg)
//	}
//}

//// WriteProperty requests a property value change from the agent
//func (c *WssServerConnection) WriteProperty(
//	thingID, name string, value any, requestID string) (err error) {
//
//	msg := wssbinding.PropertyMessage{
//		ThingID:     thingID,
//		MessageType: wssbinding.MsgTypeWriteProperty,
//		Name:        name,
//		RequestID:   requestID,
//		Data:        value,
//		Timestamp:   time.Now().Format(wot.RFC3339Milli),
//	}
//	err = c._send(msg)
//	return err
//}

// NewWSSConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IServerConnection interface.
func NewWSSConnection(
	clientID string, r *http.Request, wssConn *websocket.Conn,
	messageHandler transports.ServerMessageHandler,
) *WssServerConnection {

	clcid := "WSS" + shortid.MustGenerate()

	c := &WssServerConnection{
		wssConn:        wssConn,
		connectionID:   clcid,
		clientID:       clientID,
		messageHandler: messageHandler,
		req:            r,
		lastActivity:   time.Time{},
		mux:            sync.RWMutex{},
		observations:   connections.Subscriptions{},
		subscriptions:  connections.Subscriptions{},
	}
	return c
}
