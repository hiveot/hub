package hubclient

import (
	"encoding/json"
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
// Use this when the message is delivered to the service.
// This applies to delivery, not processing.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Completed(msg *things.ThingMessage, err error) *DeliveryStatus {
	stat.Status = DeliveryCompleted
	stat.MessageID = msg.MessageID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// Failed is a simple helper that sets the message delivery status to failed with error.
// This applies to failed delivery, not processing.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Failed(msg *things.ThingMessage, err error) *DeliveryStatus {
	stat.Status = DeliveryFailed
	stat.MessageID = msg.MessageID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// Unmarshal the reply data returned in stat
// This returns an error if unmarshalling fails or false if status does not have any data
func (stat *DeliveryStatus) Unmarshal(args interface{}) (error, bool) {
	if stat.Reply == nil {
		return nil, false
	}
	err := json.Unmarshal(stat.Reply, args)
	return err, true
}

// Marshal the status update for transport
func (stat *DeliveryStatus) Marshal() []byte {
	statJson, _ := json.Marshal(stat)
	return statJson
}
