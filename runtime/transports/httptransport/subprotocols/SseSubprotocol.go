package subprotocols

import (
	"github.com/hiveot/hub/runtime/transports/sessions"
)

// SseBinding subprotocol binding
type SseBinding struct {
	cm *sessions.ConnectionManager
	sm *sessions.SessionManager
}

// NewSseBinding returns a new SSE sub-protocol binding
//
// sm is the session manager used to add new incoming sessions
func NewSseBinding(cm *sessions.ConnectionManager, sm *sessions.SessionManager) *SseBinding {
	b := &SseBinding{
		cm: cm,
		sm: sm,
	}
	return b
}
