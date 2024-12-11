package wssserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot"
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

// _send sends the websocket message to the connected client
func (c *WssServerConnection) _send(wssMsg interface{}) (err error) {

	if c.isClosed.Load() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", c.clientID)
		slog.Warn(err.Error())
	} else {
		msgJSON := c.Marshal(wssMsg)
		// websockets do not allow concurrent write
		c.mux.Lock()
		defer c.mux.Unlock()
		err = c.wssConn.WriteMessage(websocket.TextMessage, msgJSON)
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
func (c *WssServerConnection) SendNotification(
	operation string, dThingID, name string, data any) {

	wssMsg, err := OpToMessage(operation,
		dThingID, name, nil, data, "", "")

	if err != nil {
		slog.Error("SendNotification: Unknown operation. Ignored.", "op", operation)
		return
	}

	switch operation {
	case wot.HTOpUpdateTD:
		// update the TD if the client is subscribed to its events
		if c.subscriptions.IsSubscribed(dThingID, "") {
			c._send(wssMsg)
		}
	case wot.HTOpPublishEvent:
		if c.subscriptions.IsSubscribed(dThingID, name) {
			c._send(wssMsg)
		}
	case wot.HTOpUpdateProperty, wot.HTOpUpdateMultipleProperties:
		if c.observations.IsSubscribed(dThingID, name) {
			c._send(wssMsg)
		}
	default:
		slog.Error("SendNotification: Unknown notification operation",
			"op", operation,
			"thingID", dThingID,
			"to", c.clientID)
	}
}

//
//// SendError sends an error response to the client.
//func (c *WssServerConnection) SendError(
//	thingID, name string, errResponse string, requestID string) {
//
//	if requestID == "" {
//		slog.Error("SendError without requestID", "clientID", c.clientID)
//	} else {
//		slog.Warn("SendError", "clientID", c.clientID,
//			"errResponse", errResponse, "requestID", requestID)
//	}
//	msg := ErrorMessage{
//		ThingID:     thingID,
//		MessageType: MsgTypeError,
//		Title:       name + " error",
//		RequestID:   requestID,
//		Detail:      errResponse,
//		//Timestamp:   time.Now().Format(wot.RFC3339Milli),
//	}
//	_ = c._send(msg)
//}

// SendRequest sends the request to the client (agent).
// Intended to be used on clients that are agents for Things.
// If this returns an error then no request will was sent.
func (c *WssServerConnection) SendRequest(msg transports.ThingMessage) error {
	wssMsg, err := OpToMessage(msg.Operation, msg.ThingID, msg.Name, nil,
		msg.Data, msg.RequestID, msg.SenderID)
	if err != nil {
		return err
	}
	return c._send(wssMsg)
}

// SendResponse sends an action status update to the client.
// If the status is RequestFailed then output is an error, otherwise the output value
// If this returns an error then no request will was sent.
func (c *WssServerConnection) SendResponse(
	thingID, name string, output any, errResp error, requestID string) (err error) {

	if requestID == "" {
		err = fmt.Errorf("SendResponse to '%s' without requestID", c.clientID)
	} else {
		slog.Info("SendResponse - actionStatus",
			slog.String("clientID", c.clientID),
			slog.String("requestID", requestID))
	}
	if errResp != nil {
		msg := ErrorMessage{
			ThingID:     thingID,
			MessageType: MsgTypeError,
			Title:       errResp.Error(),
			RequestID:   requestID,
			Detail:      fmt.Sprintf("%v", output),
			//Timestamp:   time.Now().Format(wot.RFC3339Milli),
		}
		_ = c._send(msg)
	} else {
		msg := ActionStatusMessage{
			ThingID:     thingID,
			MessageType: MsgTypeActionStatus,
			Name:        name,
			RequestID:   requestID,
			Output:      output,
			Timestamp:   time.Now().Format(wot.RFC3339Milli),
		}
		err = c._send(msg)
	}
	return err
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
