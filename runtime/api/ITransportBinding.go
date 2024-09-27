package api

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
)

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

	// AddTDForms add Forms to the TD for communication with the digital twin
	// using this transport binding.
	// This adds the operations for reading/writing properties, events and actions
	AddTDForms(td *tdd.TD)

	// GetProtocolInfo returns information on the protocol provided by the binding.
	GetProtocolInfo() ProtocolInfo

	// InvokeAction sends a request to invoke an action to the agent with the given ID
	// This returns the delivery status and optionally an output value or an error
	//
	// Only supported on bindings that support subscriptions
	InvokeAction(agentID string, thingID string, name string, value any) (
		status string, output any, err error)

	// PublishEvent publishes an event message to all subscribers of this protocol binding
	PublishEvent(dThingID string, name, value any)

	// PublishProperty publishes a new property value to observers of the property
	PublishProperty(dThingID string, name string, value any)

	// Start the protocol binding
	//  handler is the handler that processes incoming messages
	Start(handler hubclient.MessageHandler) error

	// Stop the protocol binding
	Stop()

	// WriteProperty sends a request to write a property to the agent with the given ID
	//
	// Only supported on bindings that support subscriptions
	WriteProperty(agentID string, thingID string, name string, value any) (status string, found bool, err error)
}
