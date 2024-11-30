package wssserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/wssbinding"
	"github.com/hiveot/hub/wot/transports/connections"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type WSSMessage map[string]any

// WssServerConnection is the Hub connection instance
//
// This implements the Hub's IServerConnection interface.
type WssServerConnection struct {
	// connection ID
	connectionID string

	// clientID is the account ID of the agent or consumer
	clientID string

	// connection request remote address
	r *http.Request

	// gorilla websocket connection
	wssConn *websocket.Conn

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// handlers of incoming events and requests
	requestHandler transports.ServerMessageHandler

	isClosed atomic.Bool

	// event subscriptions and property observations
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions
}

// _error sends a websocket error message to the connected client
func (c *WssServerConnection) _error(err error, code int, correlationID string) {
	wssMsg := wssbinding.ErrorMessage{
		MessageType:   wssbinding.MsgTypePublishError,
		Title:         err.Error(),
		CorrelationID: correlationID,
		Status:        strconv.Itoa(code),
	}
	c._send(wssMsg)
}

// _send sends the websocket message to the connected client
func (c *WssServerConnection) _send(wssMsg interface{}) (
	status string, err error) {

	if !c.isClosed.Load() {
		slog.Info("_send",
			slog.String("to", c.clientID),
		)

		msgJSON, _ := jsoniter.Marshal(wssMsg)
		// websockets do not allow concurrent write
		c.mux.Lock()
		defer c.mux.Unlock()
		err = c.wssConn.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			slog.Error("_send write error", "err", err.Error())
		}
	}
	// as long as the channel exists, delivery will take place
	// FIXME: guarantee delivery
	// todo: detect race conditions; or accept the small risk of delivery to a closing connection?
	return wssbinding.ActionStatusDelivered, nil
}

// Close closes the connection and ends the read loop
func (c *WssServerConnection) Close() {
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

// GetProtocol returns the protocol used in this connection
func (c *WssServerConnection) GetProtocol() string {
	return transports.ProtocolTypeWSS
}

// InvokeAction sends the action request for the thing to the agent
// Intended to be used on clients that are things
func (c *WssServerConnection) InvokeAction(
	thingID, name string, input any, correlationID string, senderID string) (
	status string, output any, err error) {
	msg := wssbinding.ActionMessage{
		ThingID:       thingID,
		MessageType:   wssbinding.MsgTypeInvokeAction,
		Name:          name,
		CorrelationID: correlationID,
		Data:          input,
		SenderID:      senderID,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	status, err = c._send(msg)
	return status, nil, err
}

// PublishActionStatus sends an action status update to the client.
// If an error is provided this sends the error, otherwise the output value
func (c *WssServerConnection) PublishActionStatus(
	stat transports.RequestStatus, agentID string) error {

	if stat.CorrelationID == "" {
		err := fmt.Errorf("PublishActionStatus by '%s' without requestID", agentID)
		return err
	}
	// FIXME: convert from WoT to WSS acction status vocab
	msg := wssbinding.ActionStatusMessage{
		ThingID:       stat.ThingID,
		MessageType:   wssbinding.MsgTypeActionStatus,
		Name:          stat.Name,
		Status:        stat.Status,
		CorrelationID: stat.CorrelationID,
		Output:        stat.Output,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	_, err := c._send(msg)
	return err
}

// PublishEvent send an event to subscribers
func (c *WssServerConnection) PublishEvent(
	dThingID, name string, data any, correlationID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		msg := wssbinding.EventMessage{
			ThingID:       dThingID,
			MessageType:   wssbinding.MsgTypePublishEvent,
			Name:          name,
			CorrelationID: correlationID,
			Data:          data,
			Timestamp:     time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg)
	}
}

// PublishProperty publishes a new property value clients that observe it
func (c *WssServerConnection) PublishProperty(
	dThingID string, name string, data any, correlationID string, agentID string) {

	if c.observations.IsSubscribed(dThingID, name) {
		msg := wssbinding.PropertyMessage{
			ThingID:       dThingID,
			MessageType:   wssbinding.MsgTypePropertyReading,
			Name:          name,
			CorrelationID: correlationID,
			Data:          data,
			Timestamp:     time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg)
	}
}

// WriteProperty requests a property value change from the agent
func (c *WssServerConnection) WriteProperty(
	thingID, name string, value any, correlationID string, senderID string) (status string, err error) {

	msg := wssbinding.PropertyMessage{
		ThingID:       thingID,
		MessageType:   wssbinding.MsgTypeWriteProperty,
		Name:          name,
		CorrelationID: correlationID,
		Data:          value,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	status, err = c._send(msg)
	return status, err
}

// NewWSSConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IServerConnection interface.
func NewWSSConnection(
	clientID string, r *http.Request, wssConn *websocket.Conn,
	requestHandler transports.ServerMessageHandler,
) *WssServerConnection {

	clcid := "WSS" + shortid.MustGenerate()

	c := &WssServerConnection{
		wssConn:        wssConn,
		connectionID:   clcid,
		clientID:       clientID,
		requestHandler: requestHandler,
		r:              r,
		lastActivity:   time.Time{},
		mux:            sync.RWMutex{},
		observations:   connections.Subscriptions{},
		subscriptions:  connections.Subscriptions{},
	}
	return c
}
