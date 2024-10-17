package subprotocols

import (
	sessions2 "github.com/hiveot/hub/runtime/sessions"
)

// SseBinding subprotocol binding
type SseBinding struct {
	cm *sessions2.ConnectionManager
	sm *sessions2.SessionManager
}

// NewSseBinding returns a new SSE sub-protocol binding
//
// sm is the session manager used to add new incoming sessions
func NewSseBinding(cm *sessions2.ConnectionManager, sm *sessions2.SessionManager) *SseBinding {
	b := &SseBinding{
		cm: cm,
		sm: sm,
	}
	return b
}
