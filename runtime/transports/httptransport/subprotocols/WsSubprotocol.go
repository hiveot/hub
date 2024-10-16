package subprotocols

import (
	"github.com/hiveot/hub/runtime/transports/sessions"
)

// Websocket subprotocol binding
type WsBinding struct {
	cm *sessions.ConnectionManager
	sm *sessions.SessionManager
}

// NewWsBinding returns a new websocket sub-protocol binding
func NewWsBinding(cm *sessions.ConnectionManager, sm *sessions.SessionManager) *WsBinding {
	wsBinding := &WsBinding{
		cm: cm,
		sm: sm,
	}
	return wsBinding
}
