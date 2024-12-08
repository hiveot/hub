package transports

// action status
// todo: align this with progress status values from WoT spec (if there is any)
const (
	// The request has not yet been delivered
	StatusPending = "pending"
	// The request is being processed
	StatusRunning = "running"
	// The request processing was completed
	StatusCompleted = "completed"
	// The request processing or delivery failed
	StatusFailed = "failed"
)

// RequestStatus object - currently used for all bindings unless they specify their own
// Used to send ActionStatus responses to queryaction
type RequestStatus struct {
	// ThingID of the thing reports the progress
	ThingID string `json:"thingID"`
	// The name of the thing action or property (write)
	Name string `json:"name"`
	// The correlation ID to send the response to (reply-to)
	RequestID string `json:"requestId"`
	// Status of the operation progress as per above constants: RequestPending, ...
	Status string `json:"status"`
	// Error in case delivery or processing has failed
	Error string `json:"error,omitempty"`
	// Output as per affordance, of the operation in case status is RequestCompleted
	Output any `json:"reply,omitempty"`
	//
	TimeRequested string `json:"timeRequested,omitempty"`
	TimeEnded     string `json:"timeEnded,omitempty"`
}

//
//// Completed sets the progress as completed. No more messages are send after this.
//// Optionally provide an error if it failed during processing by the Thing
////
//// msg is the internal thing message containing the request that completed.
//func (stat *RequestStatus) Completed(msg *ThingMessage, reply any, err error) *RequestStatus {
//	stat.Status = RequestCompleted
//	stat.RequestID = msg.RequestID
//	stat.Output = reply
//	stat.ThingID = msg.ThingID
//	stat.Name = msg.Name
//	if err != nil {
//		stat.Error = err.Error()
//	} else {
//		stat.Error = ""
//	}
//	return stat
//}
//
//// Delivered sets the progress to delivered (to thing agent) using the internal message.
//// The agent will update the progress to completed or failed if it set the 'rpc' flag in the
//// TD action affordance.
////
//// After delivery to the agent, the progress can still fail if the agent is unable
//// to deliver the action request to the Thing itself. If execution fails this typically
//// returns the completed status with an error.
////
//// msg is the internal thing message containing the action request that was delivered.
//func (stat *RequestStatus) Delivered(msg *ThingMessage) *RequestStatus {
//	stat.ThingID = msg.ThingID
//	stat.Name = msg.Name
//	stat.Status = RequestDelivered
//	stat.RequestID = msg.RequestID
//	return stat
//}
//
//// Failed sets the action process to failed. No more updates are sent after this.
//// This is intended to indicate a failure to deliver the action to the Thing itself.
////
////	msg is the internal thing message containing the action request that failed.
////	err contains the cause of the failure.
//func (stat *RequestStatus) Failed(msg *ThingMessage, err error) *RequestStatus {
//	stat.ThingID = msg.ThingID
//	stat.Name = msg.Name
//	stat.Status = RequestFailed
//	stat.RequestID = msg.RequestID
//	stat.Error = err.Error()
//	return stat
//}
//
//// Decode converts the native type into the given data type
//func (stat *RequestStatus) Decode(reply interface{}) (error, bool) {
//	if stat.Output == nil {
//		return nil, false
//	}
//	err := utils.Decode(stat.Output, reply)
//	return err, true
//}
