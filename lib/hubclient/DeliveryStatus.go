package hubclient

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
)

// DeliveryStatus holds the progress of action request delivery
type DeliveryStatus struct {
	// Request ID
	MessageID string `json:"messageID"`
	// Updated delivery progress
	Progress string `json:"progress"`
	// Error in case delivery or processing has failed
	Error string `json:"error"`
	// Serialized reply in case delivery and processing has completed
	Reply any `json:"reply"`
}

// Completed is a simple helper that sets the message delivery progress to completed,
// associates the message and sets the reply data or error.
//
// Use this when the processing has been completed without or with error
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Completed(msg *ThingMessage, reply any, err error) *DeliveryStatus {
	stat.Progress = vocab.ProgressStatusCompleted
	stat.MessageID = msg.MessageID
	stat.Reply = reply
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// Delivered is a simple helper that sets the message delivery progress to delivered,
// associates the message and sets the reply data or error.
//
// Use this when the request is delivered to the Thing
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Delivered(msg *ThingMessage) *DeliveryStatus {
	stat.Progress = vocab.ProgressStatusDelivered
	stat.MessageID = msg.MessageID
	return stat
}

// Failed is a simple helper that sets the message delivery status to failed with error.
//
// Use this if the messages cannot be delivered to the final destination.
// Primarily intended to make sure the messageID is not forgotten.
func (stat *DeliveryStatus) Failed(msg *ThingMessage, err error) *DeliveryStatus {
	stat.Progress = vocab.ProgressStatusFailed
	stat.MessageID = msg.MessageID
	stat.Error = err.Error()
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
