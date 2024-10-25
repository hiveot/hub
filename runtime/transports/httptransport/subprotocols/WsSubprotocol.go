package subprotocols

import (
	"github.com/hiveot/hub/runtime/connections"
)

// Websocket subprotocol binding
type WsBinding struct {
	cm *connections.ConnectionManager
}

// NewWsBinding returns a new websocket sub-protocol binding
func NewWsBinding(cm *connections.ConnectionManager) *WsBinding {
	wsBinding := &WsBinding{
		cm: cm,
	}
	return wsBinding
}
