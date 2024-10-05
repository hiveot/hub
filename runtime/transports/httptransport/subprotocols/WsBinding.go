package subprotocols

import (
	"fmt"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/runtime/transports/httptransport/sessions"
	"net/http"
)

// Websocket subprotocol binding
type WsBinding struct {
	sm *sessions.SessionManager
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

// InvokeAction sends the action request for the thing to the agent
func (b *WsBinding) InvokeAction(
	agentID, thingID, name string, data any, messageID string) (
	status string, output any, err error) {
	return digitwin.StatusFailed, nil, fmt.Errorf("Not yet implemented")
}

// PublishEvent send an event to subscribers
func (b *WsBinding) PublishEvent(dThingID, name string, data any, messageID string) {
}

// PublishProperty send a property change update to subscribers
func (b *WsBinding) PublishProperty(dThingID, name string, data any, messageID string) {
}

// WriteProperty sends the request to update the thing to the agent
func (b *WsBinding) WriteProperty(
	agentID, thingID, name string, data any, messageID string) (
	status string, err error) {

	return digitwin.StatusFailed, fmt.Errorf("Not yet implemented")
}

// NewWsBinding returns a new websocket sub-protocol binding
func NewWsBinding(sm *sessions.SessionManager) *WsBinding {
	wsBinding := &WsBinding{
		sm: sm,
	}
	return wsBinding
}
