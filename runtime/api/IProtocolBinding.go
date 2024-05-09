package api

import (
	"github.com/hiveot/hub/lib/things"
)

// Delivery status values
const (
	// DeliveryPending the request is pending delivery to the Thing's agent
	DeliveryPending = "pending"
	// DeliveryDelivered the request is delivered to the Thing's agent
	DeliveryDelivered = "delivered"
	// DeliveryWaiting the request is delivered but waiting to be applied by the agent (device could be asleep)
	DeliveryWaiting = "waiting"
	// DeliveryApplied the request has been applied to the Thing by the agent
	DeliveryApplied = "applied"
	// DeliveryCompleted the request has been applied and confirmed by the agent
	DeliveryCompleted = "completed"
	// DeliveryFailed the request failed. See DeliveryStatus Error field for more information
	DeliveryFailed = "failed"
)

// DeliveryStatus holds the progress of action request delivery
type DeliveryStatus struct {
	// Request ID
	MessageID string `json:"messageID"`
	// Updated delivery status
	Status string `json:"progress"`
	// Error in case delivery status is failed
	Error string `json:"error,omitempty"`
	// Reply in case delivery status is completed
	Reply []byte `json:"reply,omitempty"`
}

// EventHandler processes an event without return value
type EventHandler func(msg *things.ThingMessage)

// MessageHandler defines the method that processes an action or event message and
// returns a delivery status.
//
// As actions are targeted to an agent, the delivery status is that of delivery	to the agent.
// As events are broadcast, the delivery status is that of delivery to at least one subscriber.
type MessageHandler func(msg *things.ThingMessage) DeliveryStatus

// ProtocolInfo contains information provided by the binding
type ProtocolInfo struct {
	BaseURL string `json:"baseURL"`
	// The schema used by the protocol: "https, mqtt, nats"
	Schema string `json:"schema"`
	// Transport used by this protocol: "https, mqtt, nats, ..."
	// Transport IDs uniquely identify the transport: https, mqtts, nats
	Transport string `json:"transport"`
}

// IProtocolBinding is the interface implemented by all protocol bindings
type IProtocolBinding interface {

	// GetProtocolInfo returns information on the protocol provided by the binding.
	GetProtocolInfo() ProtocolInfo

	// SendToClient sends a message to a connected agent or consumer client.
	//
	// This returns a delivery status, and a reply if delivery is completed.
	// If delivery is not completed then a status update will be returned asynchronously
	// through a 'EventTypeDelivery' event.
	SendToClient(clientID string, msg *things.ThingMessage) DeliveryStatus

	// SendEvent publishes an event message to all subscribers of this protocol binding
	SendEvent(event *things.ThingMessage) DeliveryStatus

	// Start the protocol binding
	//  handler is the handler that processes incoming messages
	Start(handler MessageHandler) error

	// Stop the protocol binding
	Stop()
}
