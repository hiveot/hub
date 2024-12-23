package wssserver

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/servers/httpserver/httpcontext"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
)

// WssTransportServer Websocket subprotocol binding
type WssTransportServer struct {
	wssPath       string
	httpTransport *httpserver.HttpTransportServer
	cm            *connections.ConnectionManager

	handleRequest      transports.ServerRequestHandler
	handleResponse     transports.ServerResponseHandler
	handleNotification transports.ServerNotificationHandler

	// convert operation to message type (for building forms)
	op2MsgType map[string]string
	// opList to include in TDs
	opList []string
}

// GetConnectURL returns base path of the server with the wss connection path
func (svc *WssTransportServer) GetConnectURL() string {
	baseURL := svc.httpTransport.GetConnectURL()
	parts, _ := url.Parse(baseURL)
	wssURL, _ := url.JoinPath("wss://", parts.Host, svc.wssPath)
	return wssURL
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *WssTransportServer) SendNotification(notif transports.NotificationMessage) {
	cList := svc.cm.GetConnectionByProtocol(transports.ProtocolTypeWSS)
	for _, c := range cList {
		c.SendNotification(notif)
	}
}

// SendRequest sends a request (action, write property) to the connecting agent.
func (svc *WssTransportServer) SendRequest(req transports.RequestMessage) {
	cList := svc.cm.GetConnectionByProtocol(transports.ProtocolTypeWSS)
	for _, c := range cList {
		_ = c.SendRequest(req)
	}
}

// SendResponse send a response message to the client as a reply to a previous request.
func (svc *WssTransportServer) SendResponse(resp transports.ResponseMessage) {
	cList := svc.cm.GetConnectionByProtocol(transports.ProtocolTypeWSS)
	for _, c := range cList {
		_ = c.SendResponse(resp)
	}
}

// Serve a new websocket connection.
// This creates an instance of the WSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
func (svc *WssTransportServer) Serve(w http.ResponseWriter, r *http.Request) {
	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE connections are blocked.
	clientID, err := httpcontext.GetClientIdFromContext(r)

	if err != nil {
		slog.Warn("WS HandleConnect. No session available yet, telling client to delay retry to 10 seconds",
			"remoteAddr", r.RemoteAddr)
		errMsg := "no auth session available. Login first."
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// upgrade and validate the connection
	var upgrader = websocket.Upgrader{} // use default options
	wssConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("WS HandleConnect. Connection upgrade failed",
			"clientID", clientID, "err", err.Error())
		return
	}

	c := NewWSSConnection(clientID, r, wssConn,
		svc.handleRequest, svc.handleResponse, svc.handleNotification)

	err = svc.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	svc.ReadLoop(r.Context(), wssConn, c.WssServerHandleMessage)

	// if this fails then the connection is already closed (CloseAll)
	err = wssConn.Close()
	_ = err
	// finally cleanup the connection
	svc.cm.RemoveConnection(c.GetConnectionID())
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (svc *WssTransportServer) ReadLoop(ctx context.Context, wssConn *websocket.Conn, onMessage func([]byte)) {

	var readLoop atomic.Bool
	readLoop.Store(true)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WSSReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			readLoop.Store(false)
		}
	}()
	// read messages from the client until the connection closes
	for readLoop.Load() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			readLoop.Store(false)
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to free up the socket
		go onMessage(raw)
	}
}

func (svc *WssTransportServer) Stop() {
	// nothing to do here as this runs on top of the http server
}

// StartWssTransportServer starts a new websocket sub-protocol binding
// and attaches it to the http binding.
//
//	requestHandler receives event and request messages
//	cm is the connection registry for sending messages to clients
//	wssPath to use, without the host
//	httpTransport to attach to
func StartWssTransportServer(wssPath string, cm *connections.ConnectionManager,
	handleRequest transports.ServerRequestHandler,
	handleResponse transports.ServerResponseHandler,
	handleNotification transports.ServerNotificationHandler,
	httpTransport *httpserver.HttpTransportServer) *WssTransportServer {

	if wssPath == "" {
		wssPath = httpserver.DefaultWSSPath
	}
	// initialize the message type to operation conversion
	op2MsgType := make(map[string]string)
	opList := make([]string, 0, len(MsgTypeToOp))
	for msgType, op := range MsgTypeToOp {
		op2MsgType[op] = msgType
		opList = append(opList, op)
	}
	b := &WssTransportServer{
		cm:                 cm,
		httpTransport:      httpTransport,
		wssPath:            wssPath,
		op2MsgType:         op2MsgType,
		opList:             opList,
		handleRequest:      handleRequest,
		handleResponse:     handleResponse,
		handleNotification: handleNotification,
	}
	// add the WSS routes
	httpTransport.AddGetOp(nil, WSSOpConnect, wssPath, b.Serve)
	//httpTransport.AddGetOp(nil, WSSOpPing, wssPath, b.Serve)

	return b
}
