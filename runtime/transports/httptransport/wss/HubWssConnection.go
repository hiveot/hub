package wss

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/wssclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/connections"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type WSSMessage map[string]any

// HubWssConnection is the Hub connection instance
//
// This implements the Hub's IClientConnection interface.
type HubWssConnection struct {
	// connection ID
	connectionID string

	// clientID is the account ID of the agent or consumer
	clientID string

	// connection remote address
	remoteAddr string
	// gorilla websocket connection
	wssConn *websocket.Conn

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// digitwin router for handling of messages and requests
	dtwRouter api.IDigitwinRouter

	isClosed atomic.Bool

	subscriptions connections.Subscriptions
}

// _send sends the websocket message to the connected client
func (c *HubWssConnection) _send(
	wssMsg interface{}, correlationID string) (status string, err error) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		slog.Info("_send",
			slog.String("to", c.clientID),
			slog.String("correlationID", correlationID),
		)

		msgJSON, _ := jsoniter.Marshal(wssMsg)
		err = c.wssConn.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			slog.Error("_send write error", "err", err.Error())
		}
	}
	// as long as the channel exists, delivery will take place
	// FIXME: guarantee delivery
	// todo: detect race conditions; or accept the small risk of delivery to a closing connection?
	return wssclient.ActionStatusDelivered, nil
}

// Close closes the connection and ends the read loop
func (c *HubWssConnection) Close() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		c.isClosed.Store(true)
		_ = c.wssConn.Close()
	}
}

// GetConnectionID returns the client's unique connection ID
func (c *HubWssConnection) GetConnectionID() string {
	return c.connectionID
}

// GetClientID returns the client's account ID
func (c *HubWssConnection) GetClientID() string {
	return c.clientID
}

// InvokeAction sends the action request for the thing to the agent
// Intended to be used on clients that are things
func (c *HubWssConnection) InvokeAction(
	thingID, name string, input any, correlationID string, senderID string) (
	status string, output any, err error) {
	msg := wssclient.ActionMessage{
		ThingID:       thingID,
		MessageType:   wssclient.MsgTypeInvokeAction,
		Name:          name,
		CorrelationID: correlationID,
		Data:          input,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	status, err = c._send(msg, correlationID)
	return status, nil, err
}

// PublishActionStatus sends an action status update to the client.
// If an error is provided this sends the error, otherwise the output value
func (c *HubWssConnection) PublishActionStatus(stat hubclient.RequestStatus, agentID string) error {
	if stat.CorrelationID == "" {
		err := fmt.Errorf("PublishActionStatus by '%s' without requestID", agentID)
		return err
	}
	// FIXME: convert from WoT to WSS acction status vocab
	msg := wssclient.ActionStatusMessage{
		ThingID:       stat.ThingID,
		MessageType:   wssclient.MsgTypeActionStatus,
		Name:          stat.Name,
		Status:        stat.Status,
		CorrelationID: stat.CorrelationID,
		Output:        stat.Output,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	_, err := c._send(msg, stat.CorrelationID)
	return err
}

// PublishEvent send an event to subscribers
func (c *HubWssConnection) PublishEvent(
	dThingID, name string, data any, correlationID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		msg := wssclient.EventMessage{
			ThingID:       dThingID,
			MessageType:   wssclient.MsgTypePublishEvent,
			Name:          name,
			CorrelationID: correlationID,
			Data:          data,
			Timestamp:     time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg, correlationID)
	}
}

// PublishProperty publishes a new property value clients that observe it
func (c *HubWssConnection) PublishProperty(
	dThingID string, name string, data any, correlationID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		msg := wssclient.PropertyMessage{
			ThingID:       dThingID,
			MessageType:   wssclient.MsgTypePropertyReading,
			Name:          name,
			CorrelationID: correlationID,
			Data:          data,
			Timestamp:     time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg, correlationID)
	}
}

// SubscribeEvent adds an event subscription for this client. Use "" for wildcard
//func (c *HubWssConnection) SubscribeEvent(dThingID, name string) {
//
//}
//
//// ObserveProperty adds a property subscription for this client. Use "" for wildcard
//func (c *HubWssConnection) ObserveProperty(dThingID, name string) {
//
//}
//
//// UnsubscribeEvent removes an event subscription for this client. Use "" for wildcard
//func (c *HubWssConnection) UnsubscribeEvent(dThingID, name string) {
//
//}
//
//// UnobserveProperty removes a property subscription from this client. Use "" for wildcard
//func (c *HubWssConnection) UnobserveProperty(dThingID, name string) {
//
//}

// WriteProperty requests a property value change from the agent
func (c *HubWssConnection) WriteProperty(
	thingID, name string, value any, requestID string, senderID string) (status string, err error) {
	return wssclient.ActionStatusFailed, fmt.Errorf("not implemented")
}

// NewWSSConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IClientConnection interface.
func NewWSSConnection(
	clientID string, remoteAddr string, wssConn *websocket.Conn,
	dtwRouter api.IDigitwinRouter) *HubWssConnection {
	clcid := "WSS" + shortid.MustGenerate()

	c := &HubWssConnection{
		wssConn:       wssConn,
		connectionID:  clcid,
		clientID:      clientID,
		dtwRouter:     dtwRouter,
		remoteAddr:    remoteAddr,
		lastActivity:  time.Time{},
		mux:           sync.RWMutex{},
		subscriptions: connections.Subscriptions{},
	}
	return c
}
