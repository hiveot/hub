package subprotocols

import "net/http"

type WsBinding struct {
}

// HandleObserveProperty adds the subscription for a property's change messages
func (b *WsBinding) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleObserveAllProperties adds the subscription for sending property change messages
func (b *WsBinding) HandleObserveAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnobserveProperty removes the subscription
func (b *WsBinding) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnobserveAllProperties removes the subscription
func (b *WsBinding) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeEvent adds the subscription for a property's change messages
func (b *WsBinding) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleSubscribeAllEvents adds subscription for sending property change messages
func (b *WsBinding) HandleSubscribeAllEvents(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnsubscribeEvent removes the subscription
func (b *WsBinding) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	// todo
}

// HandleUnsubscribeAllProperties removes the subscription
func (b *WsBinding) HandleUnsubscribeAllProperties(w http.ResponseWriter, r *http.Request) {
	// todo
}
