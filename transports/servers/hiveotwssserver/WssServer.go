package hiveotwssserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
)

//
//// DefaultHiveOTWSSPath is the default path for establishing the hiveot websocket connection
//const DefaultHiveOTWSSPath = "/hiveot/wss"
//
//// DefaultHiveOTWSSPath is the default path for the WoT websocket connection
//const DefaultWoTWSSPath = "/wot/wss"

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
	// Authenticate requests
	//authenticator transports.IAuthenticator

	// The http server to register the endpoints with
	httpTransport *httpserver.HttpTransportServer

	// registered handler of incoming connections
	serverConnectHandler transports.ConnectionHandler

	// registered handler of incoming requests (which return a reply)
	serverRequestHandler transports.RequestHandler
	// registered handler of incoming responses (which sends a reply to the request sender)
	serverResponseHandler transports.ResponseHandler

	// Conversion between request/response messages and protocol messages.
	messageConverter transports.IMessageConverter

	// mutex for updating connections
	mux sync.RWMutex

	// manage the incoming connections
	connections *connections.ConnectionManager

	wssPath string
}

// AddTDForms adds forms for use of this protocol to the given TD
func (svc *WssServer) AddTDForms(tdi *td.TD) error {

	// 1 form for all operations
	form := td.Form{}
	form["op"] = "*"
	form["subprotocol"] = "wss-hiveot"
	form["contentType"] = "application/json"
	form["href"] = svc.wssPath
	tdi.Forms = append(tdi.Forms, form)
	return nil
}

func (svc *WssServer) CloseAll() {
	svc.connections.CloseAll()
}

// CloseAllClientConnections close all connections from the given client.
// Intended to close connections after a logout.
func (svc *WssServer) CloseAllClientConnections(clientID string) {
	svc.connections.ForEachConnection(func(c transports.IServerConnection) {
		if c.GetClientID() == clientID {
			c.Disconnect()
		}
	})
}

// GetConnectURL returns websocket connection URL of the server
func (svc *WssServer) GetConnectURL(_ string) string {
	httpURL := svc.httpTransport.GetConnectURL()
	parts, _ := url.Parse(httpURL)
	wssURL := fmt.Sprintf("wss://%s%s", parts.Host, svc.wssPath)
	return wssURL
}

// GetConnectionByConnectionID returns the connection with the given connection ID
func (svc *WssServer) GetConnectionByConnectionID(cid string) transports.IConnection {
	return svc.connections.GetConnectionByConnectionID(cid)
}

// GetConnectionByClientID returns the connection with the given client ID
func (svc *WssServer) GetConnectionByClientID(agentID string) transports.IConnection {
	return svc.connections.GetConnectionByClientID(agentID)
}

// GetForm returns a form for the given operation
func (svc *WssServer) GetForm(operation string, thingID string, name string) *td.Form {
	// TODO: not applicable for websockets
	return nil
}

// SendNotification sends a property update or event response message to subscribers
func (svc *WssServer) SendNotification(msg *transports.ResponseMessage) {
	// pass the response to all subscribed connections
	// FIXME: track connections
	svc.connections.ForEachConnection(func(c transports.IServerConnection) {
		c.SendNotification(*msg)
	})
}

// Serve a new websocket connection.
// This creates an instance of the HiveotWSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
func (svc *WssServer) Serve(w http.ResponseWriter, r *http.Request) {
	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE connections are blocked.
	clientID, err := httpserver.GetClientIdFromContext(r)

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

	c := NewWSSServerConnection(
		clientID, r, wssConn, svc.messageConverter)
	c.SetRequestHandler(svc.serverRequestHandler)
	c.SetResponseHandler(svc.serverResponseHandler)

	err = svc.connections.AddConnection(c)
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
	svc.connections.RemoveConnection(c.GetConnectionID())
	if svc.serverConnectHandler != nil {
		svc.serverConnectHandler(false, nil, c)
	}
}

// Stop closes all connections
func (svc *WssServer) Stop() {
	svc.CloseAll()
}

// StartHiveotWssServer returns a new websocket protocol binding that utilizes
// the given http transport protocol and message converter. This can be used by
// both the hiveot websocket protocol and WoT websocket protocol.
func StartHiveotWssServer(
	//authenticator transports.IAuthenticator,
	wssPath string,
	converter transports.IMessageConverter,
	httpTransport *httpserver.HttpTransportServer,
	handleConnect transports.ConnectionHandler,
	handleRequest transports.RequestHandler,
	handleResponse transports.ResponseHandler,
) (*WssServer, error) {
	// TODO: add the option to use the WoT websocket message format

	srv := &WssServer{
		//authenticator:         authenticator,
		httpTransport:         httpTransport,
		serverConnectHandler:  handleConnect,
		serverRequestHandler:  handleRequest,
		serverResponseHandler: handleResponse,
		wssPath:               wssPath,
		messageConverter:      converter,
		connections:           connections.NewConnectionManager(),
	}
	httpTransport.AddOps(nil, nil, http.MethodGet, wssPath, srv.Serve)

	return srv, nil
}
