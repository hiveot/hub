package api

import thing "github.com/hiveot/hub/lib/things"

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {

	// SendActionToAgent sends an action request to the intended agent.
	// This returns the message reply data or an error if the destination is not available
	SendActionToAgent(agentID string, action *thing.ThingMessage) (reply []byte, err error)

	// SendEvent publishes an event message to all subscribers of this protocol binding
	SendEvent(action *thing.ThingMessage)

	// Start the protocol binding
	Start() error

	// Stop the protocol binding
	Stop()
}
