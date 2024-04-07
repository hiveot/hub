package api

import thing "github.com/hiveot/hub/lib/things"

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {

	// SendAction sends an action request to the intended destination.
	// This returns the message reply data or an error if the destination is not available
	SendAction(action *thing.ThingMessage) (reply []byte, err error)

	// SendEvent publishes an event message to all subscribers of this protocol binding
	SendEvent(action *thing.ThingMessage)

	// SendRPC sends an rpc request to the intended destination
	// This returns the message reply data or an error if the destination is not available
	SendRPC(action *thing.ThingMessage) (reply []byte, err error)

	// Start the protocol binding
	Start() error

	// Stop the protocol binding
	Stop()
}
