package hubclient

import (
	"encoding/json"
	"github.com/hiveot/hub/lib/things"
)

// Delivery progress status values. These values only apply to the delivery process
// itself. To report errors in processing, use the Error field and set the delivery
// progress to 'DeliveryCompleted'.
//
// A successful delivery flow can take these steps, where waiting and applied are optional:
// pending -> delivered -> [waiting -> [applied ->]] completed
const (
	// DeliveredToInbox the request has been received and queued by the inbox and
	// awaiting further delivery to the agent.
	// This status is sent by the Hub inbox.
	// An additional progress update from the can be expected.
	DeliveredToInbox = "inbox"

	// DeliveredToAgent the request is delivered to the Thing's agent and awaiting further
	// updates. This is set by the server when the message is handed off to the agent.
	// This status is sent by the Hub inbox and an additional progress update from
	// the agent is expected.
	DeliveredToAgent = "agent"

	// DeliveryWaiting optional step where the agent is waiting to apply it to
	// the Thing, for example when the device is asleep.
	// This status is sent by the agent.
	// An additional progress update from the agent can be expected.
	DeliveryWaiting = "waiting"

	// DeliveryApplied is a step where the request has been applied to the Thing by the
	// agent and the agent is waiting for acceptance.
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
	MessageID string `json:"messageID"`
	// Updated delivery progress
	Progress string `json:"progress"`
	// Error in case delivery or processing has failed
	Error string `json:"error,omitempty"`
	// Serialized reply in case delivery and processing has completed
	Reply string `json:"reply,omitempty"`
}

// Completed is a simple helper that sets the message delivery progress to completed.
// Use this when the message is delivered to the service.
// This applies to delivery, not processing.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Completed(msg *things.ThingMessage, err error) DeliveryStatus {
	stat.Progress = DeliveryCompleted
	stat.MessageID = msg.MessageID
	if err != nil {
		stat.Error = err.Error()
	}
	return *stat
}

// Failed is a simple helper that sets the message delivery status to failed with error.
// This applies to failed delivery, not processing.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Failed(msg *things.ThingMessage, err error) DeliveryStatus {
	stat.Progress = DeliveryFailed
	stat.MessageID = msg.MessageID
	if err != nil {
		stat.Error = err.Error()
	}
	return *stat
}

// UnmarshalReply the reply data in this status into the given data type
// This returns an error if unmarshalling fails or false if status does not have any data
func (stat *DeliveryStatus) UnmarshalReply(reply interface{}) (error, bool) {
	if stat.Reply == "" {
		return nil, false
	}
	err := json.Unmarshal([]byte(stat.Reply), reply)
	return err, true
}

// Marshal the status message itself for transport
func (stat *DeliveryStatus) Marshal() string {
	statJson, _ := json.Marshal(stat)
	return string(statJson)
}

// MarshalReply store the serialized reply data into the delivery status
func (stat *DeliveryStatus) MarshalReply(reply interface{}) error {
	replyData, err := json.Marshal(reply)
	stat.Reply = string(replyData)
	return err
}
