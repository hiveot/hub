package sse

import (
	"github.com/hiveot/hub/wot/transports/connections"
)

// SseBinding subprotocol binding
type SseBinding struct {
	cm *connections.ConnectionManager
}

// NewSseBinding returns a new SSE sub-protocol binding
//
// sm is the session manager used to add new incoming sessions
func NewSseBinding(cm *connections.ConnectionManager) *SseBinding {
	b := &SseBinding{
		cm: cm,
	}
	return b
}
