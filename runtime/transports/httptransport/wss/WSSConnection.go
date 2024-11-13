package wss

import (
	"context"
	"fmt"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	wssclient "github.com/hiveot/hub/lib/hubclient/httpwss"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type WSSMessage map[string]any

type WSSConnection struct {
	// connection ID
	clcid string

	// clientID is the account ID of the agent or consumer
	clientID string

	// connection remote address
	remoteAddr string

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// handler of messages
	handleMessage func(msg hubclient.ThingMessage)

	// channel to write messages to the client
	wssChan  chan interface{}
	isClosed atomic.Bool

	subscriptions connections.Subscriptions
}

// _send sends the action or write request for the thing to the connected client
func (c *WSSConnection) _send(wssMsg interface{}) (status string, err error) {

	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		slog.Debug("_send",
			slog.String("to", c.clientID),
		)
		c.wssChan <- wssMsg
	}
	// as long as the channel exists, delivery will take place
	// FIXME: guarantee delivery
	// todo: detect race conditions; or accept the small risk of delivery to a closing connection?
	return vocab.RequestDelivered, nil
}

// Close closes the connection and ends the read loop
func (c *WSSConnection) Close() {
	c.mux.Lock()
	defer c.mux.Unlock()
	if !c.isClosed.Load() {
		close(c.wssChan)
		c.isClosed.Store(true)
	}
}

// GetCLCID returns the clients connection ID unique within the sessions
func (c *WSSConnection) GetCLCID() string {
	return c.clcid
}

// GetClientID returns the client's account ID
func (c *WSSConnection) GetClientID() string {
	return c.clientID
}

// InvokeAction sends the action request for the thing to the agent
// Intended to be used on clients that are things
func (c *WSSConnection) InvokeAction(
	thingID, name string, input any, requestID string, senderID string) (
	status string, output any, err error) {
	msg := wssclient.InvokeActionMessage{
		ThingID:   thingID,
		Operation: vocab.WotOpInvokeAction,
		Name:      name,
		RequestID: requestID,
		Input:     input,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
	}
	status, err = c._send(msg)
	return status, nil, err
}

// PublishActionStatus sends an action status update to the client.
// If an error is provided this sends the error, otherwise the output value
func (c *WSSConnection) PublishActionStatus(stat hubclient.RequestStatus, agentID string) error {
	if stat.RequestID == "" {
		err := fmt.Errorf("PublishActionStatus by '%s' without requestID", agentID)
		return err
	}
	msg := wssclient.ActionStatusMessage{
		ThingID:   stat.ThingID,
		Operation: vocab.WotOpPublishActionStatus,
		Name:      stat.Name,
		Progress:  stat.Progress,
		RequestID: stat.RequestID,
		Output:    stat.Output,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
	}
	_, err := c._send(msg)
	return err
}

// PublishEvent send an event to subscribers
func (c *WSSConnection) PublishEvent(
	dThingID, name string, data any, requestID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		msg := wssclient.EventMessage{
			ThingID:   dThingID,
			Operation: vocab.WotOpPublishEvent,
			Name:      name,
			RequestID: requestID,
			Data:      data,
			Timestamp: time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg)
	}
}

// PublishProperty publishes a new property value clients that observe it
func (c *WSSConnection) PublishProperty(
	dThingID string, name string, data any, requestID string, agentID string) {

	if c.subscriptions.IsSubscribed(dThingID, name) {
		msg := wssclient.PropertyMessage{
			ThingID:   dThingID,
			Operation: vocab.WotOpPublishProperty,
			Name:      name,
			RequestID: requestID,
			Data:      data,
			Timestamp: time.Now().Format(utils.RFC3339Milli),
		}
		_, _ = c._send(msg)
	}
}

// Serve serves Websocket connections.
// This listens for outgoing requests
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (c *WSSConnection) Serve(w http.ResponseWriter, r *http.Request) {

	// upgrade to websockets
	wssConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("Establishing websocket connection failed",
			"clientID", c.clientID, "err", err.Error())
		return
	}
	defer wssConn.CloseNow()

	// establish a client event channel for sending messages back to the client
	c.mux.Lock()
	c.wssChan = make(chan interface{}, 1)
	c.mux.Unlock()

	slog.Debug("WSSConnection.Serve new Websocket connection",
		slog.String("clientID", c.clientID),
		slog.String("clcid", c.clcid),
		slog.String("subprotocol", wssConn.Subprotocol()),
		slog.String("remoteAddr", c.remoteAddr),
	)

	writeLoop := true

	// close the channel when the connection drops
	go func() {
		select {
		case <-r.Context().Done(): // remote client connection closed
			slog.Debug("WSSConnection: Remote client disconnected (read context)")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			c.Close()
		}
	}()

	// 	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)

	// write messages to the client until the connection closes
	for writeLoop { // sseMsg := range sseChan {
		select {
		case wssMsg, ok := <-c.wssChan: // received event
			var err error

			if !ok { // channel was closed by session
				// avoid further writes
				writeLoop = false
				// ending the read loop and returning will close the connection
				break
			}
			slog.Debug("WSSConnection: sending ws message to client",
				//slog.String("sessionID", c.sessionID),
				slog.String("clientID", c.clientID),
				slog.String("clcid", c.clcid),
			)
			ctx := context.Background()
			err = wsjson.Write(ctx, wssConn, wssMsg)
			if err != nil {
				// writing the channel failed. Disconnect.
				_ = wssConn.Close(websocket.StatusTryAgainLater,
					"Writing failed with: "+err.Error())
				break
			}
			slog.Error("WSSConnection: Error writing message",
				slog.String("SenderID", c.clientID),
				slog.String("clcid", c.clcid),
			)
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Debug("WSSConnection.Serve: connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", c.clientID),
		slog.String("clcid", c.clcid),
	)
}

// SubscribeEvent adds an event subscription for this client. Use "" for wildcard
func (c *WSSConnection) SubscribeEvent(dThingID, name string) {

}

// ObserveProperty adds a property subscription for this client. Use "" for wildcard
func (c *WSSConnection) ObserveProperty(dThingID, name string) {

}

// UnsubscribeEvent removes an event subscription for this client. Use "" for wildcard
func (c *WSSConnection) UnsubscribeEvent(dThingID, name string) {

}

// UnobserveProperty removes a property subscription from this client. Use "" for wildcard
func (c *WSSConnection) UnobserveProperty(dThingID, name string) {

}

// WriteProperty requests a property value change from the agent
func (c *WSSConnection) WriteProperty(
	thingID, name string, value any, requestID string, senderID string) (status string, err error) {
	return vocab.RequestFailed, fmt.Errorf("not implemented")
}

// NewWSSConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IClientConnection interface.
func NewWSSConnection(clientID string, remoteAddr string) *WSSConnection {
	clcid := "WSS" + shortid.MustGenerate()

	c := &WSSConnection{
		clcid:         clcid,
		remoteAddr:    remoteAddr,
		lastActivity:  time.Time{},
		mux:           sync.RWMutex{},
		subscriptions: connections.Subscriptions{},
	}
	return c
}
