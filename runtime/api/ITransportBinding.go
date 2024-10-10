package api

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
)

// API for service action handling
type ActionHandler func(consumerID string, dThingID string, actionName string, input any, messageID string) (
	status string, output any, err error)

// ProtocolInfo contains information provided by the binding
type ProtocolInfo struct {
	BaseURL string `json:"baseURL"`
	// The schema used by the protocol: "https, mqtt, nats"
	Schema string `json:"schema"`
	// Transport used by this protocol: "https, mqtt, nats, ..."
	// Transport IDs uniquely identify the transport: https, mqtts, nats
	Transport string `json:"transport"`
}

// ITransportBinding is the interface implemented by all transport protocol bindings
type ITransportBinding interface {

	// AddTDForms adds the Forms for using this protocol bindings to the provided TD.
	// This adds the operations for reading/writing properties, events and actions
	// Original forms must be removed first as they are no longer applicable.
	AddTDForms(td *tdd.TD) error

	// GetProtocolInfo returns information on the protocol provided by the binding.
	GetProtocolInfo() ProtocolInfo

	// InvokeAction requests an action on an agent's Thing.
	// This returns the delivery status and optionally an output value or an error
	//
	// Only supported on bindings that support subscriptions. Bindings that
	// do not support subscription return an error.
	//
	// This returns a delivery status and output if delivery is completed,
	// or it returns an error if the agent is not available through this binding.
	//
	//	senderID is the sender of the request
	//	agentID is that of the agent that passes the request on to the thing
	//	thingID is that of the thing as per agent (not the digital twin)
	//	name is the name of the action as per thing action affordance
	//	value is the raw value to encode and send to the thing
	//	messageID is optional ID of the action. Used in linked events and property updates.
	InvokeAction(agentID, thingID, name string, value any, messageID string, senderID string) (
		status string, output any, err error)

	// PublishActionProgress sends an action result update to a consumer.
	//
	// Intended for one-way client connections such as sse or for async actions
	// that take time to execute.
	//
	//	clientID is the connection ID of a connected consumer that requested the action
	//	messageID of the action to associate with
	//	output result value as per TD if progress is completed, or error text if failed
	//	err error if failed
	PublishActionProgress(clientID string, stat hubclient.DeliveryStatus, agentID string) (
		found bool, err error)

	// PublishEvent publishes an event message to all subscribers of this protocol binding
	//
	//	dThingID is the Thing ID of the digital twin
	//	name is the name of the event as per digital twin event affordance
	//	value is the raw event value as per event affordance data schema
	//	messageID is the optional ID of a linked action
	PublishEvent(dThingID string, name string, value any, messageID string, agentID string)

	// PublishProperty publishes a new property value to observers of the property
	//
	//	dThingID is the Thing ID of the digital twin
	//	name is the name of the property as per digital twin property affordance
	//	value is the raw property value as per property affordance data schema
	//	messageID is the optional ID of a linked action
	PublishProperty(dThingID string, name string, value any, messageID string, agentID string)

	// Start the protocol binding
	//  handler is the handler that processes incoming messages
	//Start(handler hubclient.MessageHandler) error

	// Stop the protocol binding
	//Stop()

	// WriteProperty sends a request to write a property to the agent with the given ID
	//
	// Only supported on bindings that support subscriptions
	WriteProperty(agentID string, thingID string, name string, value any, messageID string,
		senderID string) (found bool, status string, err error)
}
