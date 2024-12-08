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
	"sync/atomic"
)

const DefaultWssPath = "/wss"

// WssTransportServer Websocket subprotocol binding
type WssTransportServer struct {
	messageHandler transports.ServerMessageHandler
	wssPath        string
	httpTransport  *httpserver.HttpTransportServer
	cm             *connections.ConnectionManager

	// convert operation to message type (for building forms)
	op2MsgType map[string]string
	// opList to include in TDs
	opList []string
}

//// GetProtocolInfo returns info on the protocol supported by this binding
//func (svc *WssTransportServer) GetProtocolInfo() transports.ProtocolInfo {
//	// todo: wss protocol info?
//	return svc.httpTransport.GetProtocolInfo()
//}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *WssTransportServer) SendNotification(operation string, dThingID, name string, data any) {
	// this is needed so mqtt can broadcast once via the message bus instead all individual connections
	// tbd. An embedded mqtt server can still send per connection?
	slog.Error("todo: implement")
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

	c := NewWSSConnection(clientID, r, wssConn, svc.messageHandler)

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
		onMessage(raw)
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
func StartWssTransportServer(
	wssPath string,
	messageHandler transports.ServerMessageHandler,
	cm *connections.ConnectionManager,
	httpTransport *httpserver.HttpTransportServer,
) *WssTransportServer {
	if wssPath == "" {
		wssPath = DefaultWssPath
	}
	// initialize the message type to operation conversion
	op2MsgType := make(map[string]string)
	opList := make([]string, 0, len(MsgTypeToOp))
	for msgType, op := range MsgTypeToOp {
		op2MsgType[op] = msgType
		opList = append(opList, op)
	}
	b := &WssTransportServer{
		cm:             cm,
		messageHandler: messageHandler,
		httpTransport:  httpTransport,
		wssPath:        wssPath,
		op2MsgType:     op2MsgType,
		opList:         opList,
	}
	// add the WSS routes
	httpTransport.AddGetOp("ws-connect", wssPath, b.Serve)

	return b
}
