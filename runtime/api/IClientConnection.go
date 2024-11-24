package api

import (
	"github.com/hiveot/hub/wot/transports"
)

// IClientConnection is the interface of an incoming consumer or agent connection.
// Transport protocol bindings must implement this interface to interact with
// the digitwin router.
//
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via this API,
// or handled internally by the protocol handler if the protocol defines the
// messages for subscription.
type IClientConnection interface {

	// Close the connection.
	// It is allowed to close an already closed connection.
	Close()

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	// this is a combination of the clientid and header cid field
	GetConnectionID() string

	// GetClientID returns the authentication ID of connected agent or consumer
	GetClientID() string

	// InvokeAction invokes an action on the Thing's agent and return result if available.
	//
	// On uni-directional connections like SSE the result will be sent as a delivery
	// update event. This is non-WoT standard as WoT doesn't support this feature.
	//
	InvokeAction(thingID string, name string, input any, requestID string, senderID string) (
		status string, output any, err error)

	// ObserveProperty adds a property subscription for this client. Use "" for wildcard
	//ObserveProperty(dThingID, name string)

	// PublishActionStatus sends an action progress update to the consumer.
	// Intended for receiving RPC results over 1-way bindings such as SSE and to
	// respond to QueryAction requests.
	PublishActionStatus(stat transports.RequestStatus, agentID string) error

	// PublishEvent publishes an event message to client. If the client is not
	// subscribed to the event then the connection will ignore the event.
	//
	//	dThingID is the digital twin thingID that publishes the event
	//	name is the event name as defined in the thing's TD
	//	value is the event value as per event affordance schema
	//	requestID if the event is associated with an action
	PublishEvent(dThingID string, name string, value any, requestID string, agentID string)

	// PublishProperty publishes a new property value to this client. If the client
	// does not observe this property then the connection will ignore the update..
	//
	//	dThingID is the digital twin thingID that publishes the property value
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	requestID if the property update is associated with an action
	PublishProperty(dThingID string, name string, value any, requestID string, agentID string)

	//// SubscribeEvent instructs this connection to add an event subscription. Use "" for wildcard
	//SubscribeEvent(dThingID, name string)
	//// UnsubscribeEvent instructs this connection to removes an event subscription. Use "" for wildcard
	//UnsubscribeEvent(dThingID, name string)
	//// UnobserveProperty instructs this connection to remove a property subscription. Use "" for wildcard
	//UnobserveProperty(dThingID, name string)

	// WriteProperty requests a property value change from the agent.
	//
	// On success, the agent will publish a property update with the given requestID.
	//
	//	thingID is the thingID as known to the agent
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	requestID if the associated property update request ID.
	//	senderID is the loginID of the consumer requesting the write
	WriteProperty(thingID, name string, value any, requestID string, senderID string) (status string, err error)
}
