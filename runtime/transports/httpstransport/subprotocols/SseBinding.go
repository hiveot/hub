package subprotocols

import (
	"net/http"
)

type SseBinding struct {
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
