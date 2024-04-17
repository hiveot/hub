package api

import (
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/router"
)

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {

	// SendActionToAgent sends an action request to a connect agent.
	// This returns the message reply data or an error if the destination is not available
	SendActionToAgent(agentID string, action *thing.ThingMessage) (reply []byte, err error)

	// SendEvent publishes an event message to all subscribers of this protocol binding
	SendEvent(event *thing.ThingMessage)

	// Start the protocol binding
	//  handler to pass incoming messages
	Start(handler router.MessageHandler) error

	// Stop the protocol binding
	Stop()
}
