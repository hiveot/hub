package subprotocols

import (
	sessions2 "github.com/hiveot/hub/runtime/sessions"
)

// Websocket subprotocol binding
type WsBinding struct {
	cm *sessions2.ConnectionManager
	sm *sessions2.SessionManager
}

// NewWsBinding returns a new websocket sub-protocol binding
func NewWsBinding(cm *sessions2.ConnectionManager, sm *sessions2.SessionManager) *WsBinding {
	wsBinding := &WsBinding{
		cm: cm,
		sm: sm,
	}
	return wsBinding
}
