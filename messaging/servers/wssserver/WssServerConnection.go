package wssserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
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

	// handler for notifications sent by agents
	notificationHandlerPtr atomic.Pointer[messaging.NotificationHandler]
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
func (sc *WssServerConnection) _send(msg any) (err error) {

	if !sc.isConnected.Load() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", sc.cinfo.ClientID)
		slog.Warn(err.Error())
	} else {
		raw, _ := jsoniter.MarshalToString(msg)
		// websockets do not allow concurrent write
		sc.mux.Lock()
		defer sc.mux.Unlock()
		err = sc.wssConn.WriteMessage(websocket.TextMessage, []byte(raw))
		if err != nil {
			err = fmt.Errorf("_send write error: %s", err)
		}
	}
	return err
}

// Disconnect closes the connection and ends the read loop
func (sc *WssServerConnection) Disconnect() {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.isConnected.Load() {
		sc.onConnection(false, nil)
		_ = sc.wssConn.Close()
	}
}

// GetConnectionInfo returns the client's connection details
func (sc *WssServerConnection) GetConnectionInfo() messaging.ConnectionInfo {
	return sc.cinfo
}

// IsConnected returns the connection status
func (sc *WssServerConnection) IsConnected() bool {
	return sc.isConnected.Load()
}
func (sc *WssServerConnection) onConnection(connected bool, err error) {
	sc.isConnected.Store(connected)
	chPtr := sc.connectionHandlerPtr.Load()
	if chPtr != nil {
		(*chPtr)(connected, err, sc)
	}
}

// onMessage handles an incoming websocket message
// The message is converted into a request or response and passed on to the registered handler.
// Messages handled by the transport binding:
// - Ping
// - (Un)ObserveProperty and (Un)ObserveAllProperties
// - (Un)SubscribeEvent and (Un)SubscribeAllEvents
func (sc *WssServerConnection) onMessage(raw []byte) {
	var err error
	sc.mux.Lock()
	sc.lastActivity = time.Now()
	sc.mux.Unlock()
	var notif *messaging.NotificationMessage
	var req *messaging.RequestMessage
	var resp *messaging.ResponseMessage

	// both non-agents and agents receive responses
	notif = sc.messageConverter.DecodeNotification(raw)
	if notif == nil {
		resp = sc.messageConverter.DecodeResponse(raw)
		if resp == nil {
			req = sc.messageConverter.DecodeRequest(raw)
		}
	}

	if notif != nil {
		notif.SenderID = sc.cinfo.ClientID
		hPtr := sc.notificationHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no notification handler set",
				"clientID", sc.cinfo.ClientID,
				"operation", notif.Operation,
			)
			return
		}
		// pass the response to the registered handler
		(*hPtr)(notif)
	} else if resp != nil {
		// only agents send responses
		resp.SenderID = sc.cinfo.ClientID
		rhPtr := sc.responseHandlerPtr.Load()
		if rhPtr != nil {
			err = (*rhPtr)(resp)
		}
	} else if req != nil {
		// sender is identified by the server, not the client
		// note that this field is still useful for services that need to know the sender
		req.SenderID = sc.cinfo.ClientID
		switch req.Operation {
		case wot.HTOpPing:
			resp = req.CreateResponse("pong", nil)

		case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
			sc.subscriptions.Subscribe(req.ThingID, req.Name, req.CorrelationID)
			resp = req.CreateResponse(nil, nil)

		case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
			sc.subscriptions.Unsubscribe(req.ThingID, req.Name)
			resp = req.CreateResponse(nil, nil)

		case wot.OpObserveProperty, wot.OpObserveAllProperties:
			sc.observations.Subscribe(req.ThingID, req.Name, req.CorrelationID)
			resp = req.CreateResponse(nil, nil)

		case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
			sc.observations.Unsubscribe(req.ThingID, req.Name)
			resp = req.CreateResponse(nil, nil)
		default:
			rhPtr := sc.requestHandlerPtr.Load()
			if rhPtr != nil {
				resp = (*rhPtr)(req, sc)
			}
		}
		if resp != nil {
			err = sc.SendResponse(resp)
		}
	} else {
		slog.Warn(
			"HandleWssMessage: Message is not a notification, request or response")
	}
	if err != nil {
		slog.Warn("Error handling websocket message", "err", err.Error())
	}
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (sc *WssServerConnection) ReadLoop(ctx context.Context, wssConn *websocket.Conn) {

	//var readLoop atomic.Bool
	sc.onConnection(true, nil)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WssServerConnection.ReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			sc.onConnection(false, nil)
		}
	}()
	// read messages from the client until the connection closes
	for sc.isConnected.Load() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			sc.onConnection(false, err)
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to free up the socket
		go sc.onMessage(raw)
	}
}

// SendNotification sends a response to the client if subscribed.
// this is a response to a long-running subscription request
// If this returns an error then no response was sent.
func (sc *WssServerConnection) SendNotification(
	notif *messaging.NotificationMessage) (err error) {

	if !strings.HasPrefix(notif.ThingID, "dtw") {
		//panic("missing dtw prefix") // for testing
		// normally notifications are from digital twins with an ID containing the dtw: prefix
		slog.Error("SendNotification: ThingID has no dtw: prefix",
			"thingID", notif.ThingID, "senderID", notif.SenderID)
	}
	switch notif.Operation {
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		correlationID := sc.subscriptions.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("SendNotification (event subscription)",
				slog.String("clientID", sc.cinfo.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("event name", notif.Name),
			)
			msg, _ := sc.messageConverter.EncodeNotification(notif)
			err = sc._send(msg)
		}
	case wot.OpObserveProperty, wot.OpObserveMultipleProperties, wot.OpObserveAllProperties:
		correlationID := sc.observations.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("SendNotification (observed property(ies))",
				slog.String("clientID", sc.cinfo.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
			msg, _ := sc.messageConverter.EncodeNotification(notif)
			err = sc._send(msg)
		}
	case wot.OpInvokeAction:
		// action progress update, for original sender only
		slog.Info("SendNotification (action status)",
			slog.String("clientID", sc.cinfo.ClientID),
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
		)
		msg, _ := sc.messageConverter.EncodeNotification(notif)
		err = sc._send(msg)
	default:
		slog.Warn("Unknown notification: " + notif.Operation)
	}
	return err
}

// SendRequest sends the request to the client (agent).
//
// Intended to be used on cm that are agents for Things and connect to the hub
// as a client (connection reversal).
// If this server is the Thing agent then there is no need for this method.
//
// If this returns an error then no request was sent.
func (sc *WssServerConnection) SendRequest(req *messaging.RequestMessage) error {
	msg, err := sc.messageConverter.EncodeRequest(req)
	if err == nil {
		err = sc._send(msg)
	}
	return err
}

// SendResponse sends a response to the remote client.
// If this returns an error then no response was sent.
func (sc *WssServerConnection) SendResponse(resp *messaging.ResponseMessage) (err error) {

	//slog.Info("SendResponse (server->client)",
	//	slog.String("clientID", sc.cinfo.ClientID),
	//	slog.String("correlationID", resp.CorrelationID),
	//	slog.String("operation", resp.Operation),
	//	slog.String("name", resp.Name),
	//	slog.String("status", resp.Status),
	//	slog.String("type", resp.MessageType),
	//	slog.String("senderID", resp.SenderID),
	//)

	msg := sc.messageConverter.EncodeResponse(resp)
	err = sc._send(msg)
	return err
}

func (sc *WssServerConnection) SetConnectHandler(cb messaging.ConnectionHandler) {
	if cb == nil {
		sc.connectionHandlerPtr.Store(nil)
	} else {
		sc.connectionHandlerPtr.Store(&cb)
	}
}

// SetNotificationHandler set the application handler for received notifications
func (sc *WssServerConnection) SetNotificationHandler(cb messaging.NotificationHandler) {
	if cb == nil {
		sc.notificationHandlerPtr.Store(nil)
	} else {
		sc.notificationHandlerPtr.Store(&cb)
	}
}
func (sc *WssServerConnection) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		sc.requestHandlerPtr.Store(nil)
	} else {
		sc.requestHandlerPtr.Store(&cb)
	}
}
func (sc *WssServerConnection) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		sc.responseHandlerPtr.Store(nil)
	} else {
		sc.responseHandlerPtr.Store(&cb)
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
		//ProtocolType: messageConverter.GetProtocolType(),
		Timeout: 0,
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
