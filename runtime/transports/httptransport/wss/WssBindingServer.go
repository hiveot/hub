package wss

import (
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/transports/httptransport/httpcontext"
	"github.com/hiveot/hub/wot/transports/clients/wssbinding"
	"log/slog"
	"net/http"
)

// HubWssBinding Websocket subprotocol binding
type HubWssBinding struct {
	cm        *connections.ConnectionManager
	dtwRouter api.IDigitwinRouter
}

// Serve a new websocket connection.
// This creates an instance of the WSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
func (b *HubWssBinding) Serve(w http.ResponseWriter, r *http.Request) {
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

	c := NewWSSConnection(clientID, r.RemoteAddr, wssConn, b.dtwRouter)

	err = b.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	wssbinding.WSSReadLoop(r.Context(), wssConn, c.HubWssReceive)

	// if this fails then the connection is already closed (CloseAll)
	err = wssConn.Close()
	_ = err
	// finally cleanup the connection
	b.cm.RemoveConnection(c.GetConnectionID())
}

// NewWssBinding returns a new websocket sub-protocol binding
func NewWssBinding(
	cm *connections.ConnectionManager, dtwRouter api.IDigitwinRouter) *HubWssBinding {
	wsBinding := &HubWssBinding{
		cm:        cm,
		dtwRouter: dtwRouter,
	}
	return wsBinding
}
