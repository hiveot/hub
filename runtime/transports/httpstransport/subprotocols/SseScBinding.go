package subprotocols

import "net/http"

type SseScBinding struct {
}

// HandleObserveProperty adds a property subscription
func (b *SseScBinding) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleObserveAllProperties adds a property subscription
func (b *SseScBinding) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnobserveProperty removes the subscription
func (b *SseScBinding) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnobserveAllProperties removes the subscription
func (b *SseScBinding) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeEvent adds an event subscription
func (b *SseScBinding) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeAllEvents adds a subscription to all events
func (b *SseScBinding) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnsubscribeEvent removes the subscription
func (b *SseScBinding) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnsubscribeAllEvents removes the subscription
func (b *SseScBinding) HandleUnsubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	// todo
}
