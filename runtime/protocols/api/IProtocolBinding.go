package api

import thing "github.com/hiveot/hub/lib/things"

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {

	// Publish a message to subscribers of this protocol binding
	// The protocol binding subscription logic ensures that only authorized clients will receive the messsage.
	Publish(message *thing.ThingValue)

	// Start the protocol binding
	Start() error

	// Stop the protocol binding
	Stop()
}
