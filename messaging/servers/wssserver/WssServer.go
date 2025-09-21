package wssserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/messaging/servers/httpbasic"
)

const (
	DefaultWssPath = "/wot/wss"
	SubprotocolWSS = "websocket"
	WssSchema      = "wss"
)

// WssServer is a websocket transport protocol server for use with HiveOT and WoT
// messages.
//
// Use AddEndpoint to add a service endpoint to listen on and a corresponding message converter.
//
// While intended for the Hub, it can also be used in stand-alone Things that
// run their own servers. An https server is required.
//
// The difference with the WoT Websocket protocol is that it transport the Request
// and Response messages directly as-is, using JSON encoding.
//
// Connections support event subscription and property observe requests, and sends
// updates as Responses with the subscription correlationID.
type WssServer struct {

	// manage the incoming connections
	cm *connections.ConnectionManager

	// the connection URL for this websocket server
	connectURL string

	// The router to register with
	router chi.Router

	// registered handler to notify of incoming connections
	serverConnectHandler messaging.ConnectionHandler

	// registered handler of incoming notifications
	serverNotificationHandler messaging.NotificationHandler
	// registered handler of incoming requests (which return a reply)
	serverRequestHandler messaging.RequestHandler
	// registered handler of incoming responses (which sends a reply to the request sender)
	serverResponseHandler messaging.ResponseHandler

	// Conversion between request/response messages and protocol messages.
	messageConverter messaging.IMessageConverter

	// mutex for updating cm
	mux sync.RWMutex

	// listening path for incoming connections
	wssPath string
}

func (srv *WssServer) CloseAll() {
	srv.cm.CloseAll()
}

// CloseAllClientConnections close all cm from the given client.
// Intended to close cm after a logout.
func (srv *WssServer) CloseAllClientConnections(clientID string) {
	srv.cm.ForEachConnection(func(c messaging.IServerConnection) {
		cinfo := c.GetConnectionInfo()
		if cinfo.ClientID == clientID {
			c.Disconnect()
		}
	})
}

// GetConnectURL returns websocket connection URL of the server
func (srv *WssServer) GetConnectURL() string {
	return srv.connectURL
}

// GetConnectionByConnectionID returns the connection with the given connection ID
func (srv *WssServer) GetConnectionByConnectionID(clientID, cid string) messaging.IConnection {
	return srv.cm.GetConnectionByConnectionID(clientID, cid)
}

// GetConnectionByClientID returns the connection with the given client ID
func (srv *WssServer) GetConnectionByClientID(agentID string) messaging.IConnection {
	return srv.cm.GetConnectionByClientID(agentID)
}

// GetProtocolType returns the protocol type of this server
func (srv *WssServer) GetProtocolType() string {
	return messaging.ProtocolTypeWSS
}

// SendNotification sends a property update or event response message to subscribers
func (srv *WssServer) SendNotification(msg *messaging.NotificationMessage) {
	// pass the response to all subscribed cm
	srv.cm.ForEachConnection(func(c messaging.IServerConnection) {
		_ = c.SendNotification(msg)
	})
}

// Serve a new websocket connection.
// This creates an instance of the HiveotWSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
//
// serverRequestHandler and serverResponseHandler are used as handlers for incoming
// messages.
func (srv *WssServer) Serve(w http.ResponseWriter, r *http.Request) {
	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE cm are blocked.
	clientID, err := httpbasic.GetClientIdFromContext(r)

	slog.Info("Receiving Websocket connection", slog.String("clientID", clientID))

	if err != nil {
		slog.Warn("Serve. No clientID",
			"remoteAddr", r.RemoteAddr)
		errMsg := "no auth session available. Login first."
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// upgrade and validate the connection
	var upgrader = websocket.Upgrader{} // use default options
	wssConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("Serve. Connection upgrade failed",
			"clientID", clientID, "err", err.Error())
		return
	}

	c := NewWSSServerConnection(clientID, r, wssConn, srv.messageConverter)
	c.SetNotificationHandler(srv.serverNotificationHandler)
	c.SetRequestHandler(srv.serverRequestHandler)
	c.SetResponseHandler(srv.serverResponseHandler)

	err = srv.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.ReadLoop(r.Context(), wssConn)

	// if this fails then the connection is already closed (CloseAll)
	err = wssConn.Close()
	_ = err
	// finally cleanup the connection
	srv.cm.RemoveConnection(c)
	if srv.serverConnectHandler != nil {
		srv.serverConnectHandler(false, nil, c)
	}
}

// Start listening for incoming SSE connections
func (srv *WssServer) Start() error {
	slog.Info("Starting websocket server, Listening on: " + srv.GetConnectURL())
	// TODO: detect if already listening
	srv.router.Get(srv.wssPath, srv.Serve)
	return nil
}

// Stop disconnects clients and remove connection listening
func (srv *WssServer) Stop() {
	srv.CloseAll()
	srv.router.Delete(srv.wssPath, srv.Serve)
}

// NewWssServer returns a new websocket protocol server. Use Start() to activate routes.
//
// The given message converter maps between the underlying websocket message and the
// hiveot Request/ResponseMessage envelopes.
//
// connectAddr is the host:port of the webserver
// wsspath is the path of the websocket endpoint that will listen on the server
// router is the protected route that serves websocket on the wssPath
// converter converts between the internal message format and the webscoket message protocol
// handleConnect optional callback for each websocket connection
// handleNotification optional callback to invoke when a notification message is received
// handleRequesst optional callback to invoke when a request message is received
// handleResponse optional callback to invoke when a response message is received
func NewWssServer(
	connectAddr string,
	wssPath string,
	router chi.Router,
	converter messaging.IMessageConverter,
	handleConnect messaging.ConnectionHandler,
	handleNotification messaging.NotificationHandler,
	handleRequest messaging.RequestHandler,
	handleResponse messaging.ResponseHandler,
) *WssServer {

	connectURL := fmt.Sprintf("%s://%s%s", WssSchema, connectAddr, wssPath)
	srv := &WssServer{
		connectURL:                connectURL,
		serverConnectHandler:      handleConnect,
		serverNotificationHandler: handleNotification,
		serverRequestHandler:      handleRequest,
		serverResponseHandler:     handleResponse,
		messageConverter:          converter,
		cm:                        connections.NewConnectionManager(),
		router:                    router,
		wssPath:                   wssPath,
	}
	return srv
}
