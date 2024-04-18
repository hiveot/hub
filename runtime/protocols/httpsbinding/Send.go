package httpsbinding

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// SendActionToAgent sends the action request to the agent and return the result
func (svc *HttpsBinding) SendActionToAgent(agentID string, action *things.ThingMessage) (resp []byte, err error) {
	// this requires an sse or WS connection from that agent
	return nil, fmt.Errorf("not yet implemented")
}

// SendEvent an event message to subscribers
// This passes it to SSE handlers of active sessions
func (svc *HttpsBinding) SendEvent(message *things.ThingMessage) {
	//sessions := sessionmanager.GetSessions()
	// TODO: track subscriptions
	// TODO: publish to SSE handlers of subscribed clients
}
