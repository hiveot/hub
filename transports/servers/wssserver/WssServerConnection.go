package wssserver

import (
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/teris-io/shortid"
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
	httpReq *http.Request

	// gorilla websocket connection
	wssConn *websocket.Conn

	// track last used time to auto-close inactive connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// handler for requests send by clients
	requestHandler transports.ServerRequestHandler
	// handler for responses sent by agents
	responseHandler transports.ServerResponseHandler
	// handler for notifications sent by agents
	notificationHandler transports.ServerNotificationHandler

	isClosed atomic.Bool

	// event subscriptions and property observations by consumers
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions
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

// NewWSSConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IServerConnection interface.
func NewWSSConnection(
	clientID string, r *http.Request, wssConn *websocket.Conn,
	requestHandler transports.ServerRequestHandler,
	responseHandler transports.ServerResponseHandler,
	notificationHandler transports.ServerNotificationHandler,
) *WssServerConnection {

	clcid := "WSS" + shortid.MustGenerate()

	c := &WssServerConnection{
		wssConn:             wssConn,
		connectionID:        clcid,
		clientID:            clientID,
		requestHandler:      requestHandler,
		responseHandler:     responseHandler,
		notificationHandler: notificationHandler,
		httpReq:             r,
		lastActivity:        time.Time{},
		mux:                 sync.RWMutex{},
		observations:        connections.Subscriptions{},
		subscriptions:       connections.Subscriptions{},
	}
	return c
}
