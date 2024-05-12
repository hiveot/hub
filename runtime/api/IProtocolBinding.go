package api

import (
	"github.com/hiveot/hub/lib/things"
)

// Delivery status values. These values only apply to the delivery process itself.
// To report errors in marshalling or processing use the Error field and set the
// delivery status to 'DeliveryCompleted'.
//
// A successful delivery flow can take these steps, where waiting and applied are optional:
// pending -> delivered -> [waiting -> [applied ->]] completed
const (
	// DeliveryPending the request is pending delivery to the Thing's agent.
	// This status is sent by the Hub inbox.
	// An additional status update from the hub can be expected.
	DeliveryPending = "pending"
	// DeliveryDelivered the request is delivered to the Thing's agent and awaiting further
	// updates. This is set by the server when the message is handed off to the agent.
	// This status is sent by the Hub inbox.
	// An additional status update from the agent is expected.
	DeliveryDelivered = "delivered"
	// DeliveryWaiting optional step where the request is delivered to the agent but
	// the agent is waiting to apply it to the Thing, for example when the device is asleep.
	// This status is sent by the agent.
	// An additional status update from the agent can be expected.
	DeliveryWaiting = "waiting"
	// DeliveryApplied optional step where the request has been applied to the Thing by the
	// agent and the agent is waiting for confirmation.
	// Intended for device actions that take time to complete.
	// This status is sent by the agent.
	// An additional status update from the agent can be expected.
	DeliveryApplied = "applied"
	// DeliveryCompleted the request has been received and applied by the agent and
	// all steps are completed.
	// If unmarshalling or processing fails by the client then the delivery was still
	// completed and an error is included in the delivery status.
	// This status is sent by the agent and is final. No additional status updates will be sent.
	DeliveryCompleted = "completed"
	// DeliveryFailed the request could not be delivered to the agent or the agent can
	// not deliver the request to the Thing. (if the agent is the Thing then this is the same thing)
	// Only use this to indicate a delivery problem between Hub and Thing, not for
	// processing or marshalling problems.
	// Use the error field with status DeliveryCompleted in case of processing errors.
	// This status can be sent by the Hub or the Agent.
	// This status is final. No additional status updates will be sent.
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

// Completed is a simple helper that sets the message delivery status to completed.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Completed(msg *things.ThingMessage, err error) {
	stat.Status = DeliveryCompleted
	stat.MessageID = msg.MessageID
	if err != nil {
		stat.Error = err.Error()
	}
}

// Failed is a simple helper that sets the message delivery status to failed with error.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Failed(msg *things.ThingMessage, err error) {
	stat.Status = DeliveryFailed
	stat.MessageID = msg.MessageID
	stat.Error = err.Error()
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
