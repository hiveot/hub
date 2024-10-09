package subprotocols

import "github.com/hiveot/hub/lib/hubclient"

// IClientConnection is the interface of an incoming consumer or agent connection.
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via the API
// or internally by the protocol handler if the protocol is bi-directional
// and defines messages for subscription.
type IClientConnection interface {

	// Close the connection
	Close()

	// GetClientID returns the ID of connected agent or consumer
	GetClientID() string

	// InvokeAction an action on this agent and return result if available.
	//
	// On uni-directional connections like SSE the result will be sent as a delivery
	// update event. This is non-WoT standard as WoT doesn't support this feature.
	InvokeAction(thingID string, name string, input any, messageID string) (
		status string, output any, err error)

	// PublishActionProgress sends an action result value to this consumer.
	// Intended for rpc style actions.
	//
	// Intended for receiving RPC results over 1-way bindings such as SSE.
	PublishActionProgress(stat hubclient.DeliveryStatus) error

	// PublishEvent publishes an event message to client, if subscribed
	//
	//	dThingID is the digital twin thingID that publishes the event
	//	name is the event name as defined in the thing's TD
	//	value is the event value as per event affordance schema
	//	messageID if the event is associated with an action
	PublishEvent(dThingID string, name string, value any, messageID string)

	// PublishProperty publishes a new property value clients that observe it
	//
	//	dThingID is the digital twin thingID that publishes the property value
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	messageID if the property update is associated with an action
	PublishProperty(dThingID string, name string, value any, messageID string)

	// SubscribeEvent adds an event subscription for this client. Use "" for wildcard
	SubscribeEvent(dThingID, name string)
	// ObserveProperty adds a property subscription for this client. Use "" for wildcard
	ObserveProperty(dThingID, name string)
	// UnsubscribeEvent removes an event subscription for this client. Use "" for wildcard
	UnsubscribeEvent(dThingID, name string)
	// UnobserveProperty removes a property subscription from this client. Use "" for wildcard
	UnobserveProperty(dThingID, name string)

	// WriteProperty requests a property value change from the client (agent)
	//
	// On success, the agent will publish a property update with the given messageID.
	//
	//	thingID is the thingID as known to the agent
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	messageID if the associated property update request ID.
	WriteProperty(thingID, name string, value any, messageID string) (status string, err error)
}
