package wss

import (
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/transports/httptransport/httpcontext"
	"log/slog"
	"net/http"
)

// Websocket subprotocol binding
type WssBinding struct {
	cm *connections.ConnectionManager
}

// HandleConnect handles a new websocket connection.
// This doesn't return until the connection is closed by either client or server.
func (b *WssBinding) HandleConnect(w http.ResponseWriter, r *http.Request) {
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

	// add the new WS connection
	c := NewWSSConnection(clientID, r.RemoteAddr)

	err = b.cm.AddConnection(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	b.cm.RemoveConnection(c.GetCLCID())
}

// NewWssBinding returns a new websocket sub-protocol binding
func NewWssBinding(cm *connections.ConnectionManager) *WssBinding {
	wsBinding := &WssBinding{
		cm: cm,
	}
	return wsBinding
}
