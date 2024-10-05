package hubclient

import (
	"github.com/hiveot/hub/lib/utils"
)

// Delivery progress status values. These values only apply to the delivery process
// itself. To report errors in processing, use the Error field and set the delivery
// progress to 'DeliveryCompleted'.
//
// A successful delivery flow can take these steps, where waiting and applied are optional:
// pending -> delivered -> [waiting -> [applied ->]] completed
const (
	// DeliveryPending the has been received and queued by the inbox and
	// awaiting further delivery to the agent.
	// This is the default status when the message is received by the hub.
	// An additional progress update can be expected.
	DeliveryPending = "pending"

	// DeliveryDelivered the request is delivered to the Thing's agent and awaiting further
	// updates. This is set by the server when the message is handed off to the agent.
	// This status is sent by the Hub and an additional progress update from
	// the agent is expected.
	DeliveryDelivered = "delivered"

	// DeliveryApplied is an optional step where the request has been applied to
	// the Thing by the agent and the agent is waiting for a result.
	// Intended for device actions that take time to complete.
	// This status is sent by the agent.
	// An additional progress update from the agent can be expected.
	DeliveryApplied = "applied"

	// DeliveryCompleted the request has been applied and a result received.
	// All steps are completed.
	// If unmarshalling or processing fails by the client then the delivery was still
	// completed and an error is included in the delivery progress update.
	// This status is sent by the agent and is final. No additional progress updates will be sent.
	DeliveryCompleted = "completed"

	// DeliveryFailed the request could not be delivered to the agent or the agent can
	// not apply the request to the Thing. (if the agent is the Thing then this is the same thing)
	// Only use this to indicate a delivery problem between Hub and Thing, not for
	// processing or marshalling problems.
	// Use the error field with progress DeliveryCompleted in case of processing errors.
	// This status can be sent by the Hub or the Agent.
	// This status is final. No additional progress updates will be sent.
	DeliveryFailed = "failed"
)

// DeliveryStatus holds the progress of action request delivery
type DeliveryStatus struct {
	// Request ID
	MessageID string
	// Updated delivery progress
	Progress string
	// Error in case delivery or processing has failed
	Error error
	// Serialized reply in case delivery and processing has completed
	Reply any
}

// Applied is a simple helper that sets the message delivery progress to applied,
// and associates the message.
//
// Use this  when the request has been applied to the device but not yet completed.
//
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Applied(msg *ThingMessage) *DeliveryStatus {
	stat.Progress = DeliveryApplied
	stat.MessageID = msg.MessageID
	return stat
}

// Completed is a simple helper that sets the message delivery progress to completed,
// associates the message and sets the reply data or error.
//
// # Use this when the processing has been completed without or with error
//
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Completed(msg *ThingMessage, reply any, err error) *DeliveryStatus {
	stat.Progress = DeliveryCompleted
	stat.MessageID = msg.MessageID
	stat.Reply = reply
	stat.Error = err
	return stat
}

// Failed is a simple helper that sets the message delivery status to failed with error.
//
// Use this if the messages cannot be delivered to the final destination.
//
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Failed(msg *ThingMessage, err error) *DeliveryStatus {
	stat.Progress = DeliveryFailed
	stat.MessageID = msg.MessageID
	stat.Error = err
	return stat
}

// Decode converts the native type into the given data type
func (stat *DeliveryStatus) Decode(reply interface{}) (error, bool) {
	if stat.Reply == nil {
		return nil, false
	}
	err := utils.Decode(stat.Reply, reply)
	return err, true
}
