package hubclient

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
)

// RequestProgress holds the progress of action or write property request delivery.
// Intended for RPC updates and for asynchronously receiving action progress updates.
// ThingID and Name are intended for the latter.

// TBD: can progress updates be replaced by property updates?
// A: no, property updates have no progress/error fields
//    no, stateful action progress might not have any output value?
//        stateless action progress should use rpc
//        stateful action progress should have a corresponding state property

type RequestProgress struct {
	// ThingID of the thing handles the action.
	ThingID string `json:"thingID"`
	// The action name
	Name string `json:"name"`
	// Request ID
	RequestID string `json:"requestID"`
	// Updated delivery progress
	Progress string `json:"progress"`
	// Error in case delivery or processing has failed
	Error string `json:"error"`
	// Native reply data in case delivery and processing has completed
	Reply any `json:"reply"`
}

// Completed sets the progress as completed. No more messages are send after this.
// Optionally provide an error if it failed during processing by the Thing
//
// msg is the internal thing message containing the request that completed.
func (stat *RequestProgress) Completed(msg *ThingMessage, reply any, err error) *RequestProgress {
	stat.Progress = vocab.RequestCompleted
	stat.RequestID = msg.RequestID
	stat.Reply = reply
	stat.ThingID = msg.ThingID
	stat.Name = msg.Name
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// Delivered sets the progress to delivered (to thing agent) using the internal message.
// The agent will update the progress to completed or failed if it set the 'rpc' flag in the
// TD action affordance.
//
// After delivery to the agent, the progress can still fail if the agent is unable
// to deliver the action request to the Thing itself. If execution fails this typically
// returns the completed status with an error.
//
// msg is the internal thing message containing the action request that was delivered.
func (stat *RequestProgress) Delivered(msg *ThingMessage) *RequestProgress {
	stat.ThingID = msg.ThingID
	stat.Name = msg.Name
	stat.Progress = vocab.RequestDelivered
	stat.RequestID = msg.RequestID
	return stat
}

// Failed sets the action process to failed. No more updates are sent after this.
// This is intended to indicate a failure to deliver the action to the Thing itself.
//
//	msg is the internal thing message containing the action request that failed.
//	err contains the cause of the failure.
func (stat *RequestProgress) Failed(msg *ThingMessage, err error) *RequestProgress {
	stat.ThingID = msg.ThingID
	stat.Name = msg.Name
	stat.Progress = vocab.RequestFailed
	stat.RequestID = msg.RequestID
	stat.Error = err.Error()
	return stat
}

// Decode converts the native type into the given data type
func (stat *RequestProgress) Decode(reply interface{}) (error, bool) {
	if stat.Reply == nil {
		return nil, false
	}
	err := utils.Decode(stat.Reply, reply)
	return err, true
}
