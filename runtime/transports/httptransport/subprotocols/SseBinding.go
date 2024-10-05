package subprotocols

import (
	"fmt"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/runtime/transports/httptransport/sessions"
	"net/http"
)

// SseBinding subprotocol binding
type SseBinding struct {
	sm *sessions.SessionManager
}

// HandleObserveProperty adds the sse handler for a property's change messages
func (b *SseBinding) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleObserveAllProperties adds the sse handler for sending property change messages
func (b *SseBinding) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeEvent adds the sse handler for an event subscription
func (b *SseBinding) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeAllEvents adds the sse handler for all events
func (b *SseBinding) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	// todo
}

// InvokeAction sends the action request for the thing to the agent
func (b *SseBinding) InvokeAction(
	agentID, thingID, name string, data any, messageID string) (
	status string, output any, err error) {
	return digitwin.StatusFailed, nil, fmt.Errorf("Not yet implemented")
}

// PublishEvent send an event to subscribers
func (b *SseBinding) PublishEvent(dThingID, name string, data any, messageID string) {
}

// PublishProperty send a property change update to subscribers
func (b *SseBinding) PublishProperty(dThingID, name string, data any, messageID string) {
}

// WriteProperty sends the request to update the thing to the agent
func (b *SseBinding) WriteProperty(
	agentID, thingID, name string, data any, messageID string) (
	status string, err error) {

	return digitwin.StatusFailed, fmt.Errorf("Not yet implemented")
}

// NewSseBinding returns a new SSE sub-protocol binding
//
// sm is the session manager used to add new incoming sessions
func NewSseBinding(sm *sessions.SessionManager) *SseBinding {
	b := &SseBinding{
		sm: sm,
	}
	return b
}
