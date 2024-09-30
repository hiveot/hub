package subprotocols

// IProtocolConnection is the interface of protocol connection handlers.
//
// Protocol connections provide a channel for sending messages from the
// digital twin to agents or consumers.
//
// Subscription to events or properties can be made externally via the API
// or internally by the protocol handler if the protocol is bi-directional
// and defines messages for subscription.
type IProtocolConnection interface {

	// Close the connection
	Close()

	// ID of connected agent or consumer
	GetClientID() string

	// InvokeAction an action on the agent and return result if available
	InvokeAction(agentID string, thingID string, name string, input any, messageID string) (
		status string, output any, err error)

	// PublishEvent publishes an event message to all subscribers of this protocol binding
	//
	//	dThingID is the digital twin thingID that publishes the event
	//	name is the event name as defined in the thing's TD
	//	value is the event value as per event affordance schema
	//	messageID if the event is associated with an action
	PublishEvent(dThingID string, name string, value any, messageID string)

	// PublishProperty publishes a new property value to observers of the property
	//
	//	dThingID is the digital twin thingID that publishes the property value
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	messageID if the property update is associated with an action
	PublishProperty(dThingID string, name string, value any, messageID string)

	// SubscribeEvent adds a subscription to the connection. Use "" for wildcard
	SubscribeEvent(dThingID, name string)
	// ObserveProperty adds a subscription to the connection. Use "" for wildcard
	ObserveProperty(dThingID, name string)
	// UnsubscribeEvent removes a subscription from the connection. Use "" for wildcard
	UnsubscribeEvent(dThingID, name string)
	// UnobserveProperty removes a subscription from the connection. Use "" for wildcard
	UnobserveProperty(dThingID, name string)

	// WriteProperty requests a property value change from a thing agent
	//
	// On success, the agent will publish a property update with the given messageID.
	//
	//  agentID is the login ID of the agent
	//	thingID is the thingID as known to the agent
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	messageID if the associated property update request ID.
	WriteProperty(agentID, thingID, name string, value any, messageID string) (status string, err error)
}
