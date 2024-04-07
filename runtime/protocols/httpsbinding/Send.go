package httpsbinding

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// SendAction an action message to the destination
// This returns an error if no session for the destination is available
func (svc *HttpsBinding) SendAction(message *things.ThingMessage) ([]byte, error) {
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
	return nil, fmt.Errorf("not yet implemented")
}

// SendEvent an event message to subscribers
// This passes it to SSE handlers of active sessions
func (svc *HttpsBinding) SendEvent(message *things.ThingMessage) {
	//sessions := sessionmanager.GetSessions()
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
}

// SendRPC an rpc requestmessage to the destination
// This returns an error if no session for the destination is available
func (svc *HttpsBinding) SendRPC(message *things.ThingMessage) ([]byte, error) {
	return nil, fmt.Errorf("not yet implemented")
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
}
