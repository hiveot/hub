package transports

// action status
// this aligns with action status values from WoT spec
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

// ActionStatus object - compatible with the HTTP binding but also used
// internally to exchange messages with various protocol binding. Each binding
// should convert it to its own definition as per spec. HTTP binding can use
// it in-place.
//
// This is close to the digitwin ActionValue object (from the TD) intentionally
// so it can be converted easily.
//type ActionStatus struct {
//	//--- The HTTP binding specifies these fields:
//	// Status of the operation progress as per above constants: RequestPending, ...
//	Status string `json:"status"`
//	// Error in case delivery or processing has failed
//	Error string `json:"error,omitempty"`
//	// HRef is a link to the resource
//	HRef string `json:"HRef,omitempty"`
//	// Output as per affordance, of the operation in case status is RequestCompleted
//	Output any `json:"output,omitempty"`
//	//
//	TimeRequested string `json:"timeRequested,omitempty"`
//	TimeEnded     string `json:"timeEnded,omitempty"`
//
//	//--- HiveOT adds these fields:
//	// AgentID of the agent that manages the Thing
//	AgentID string `json:"agentID"`
//	// Input of the action
//	Input any `json:"input,omitempty"`
//	// Operation is used to identify the type of action to allows it to be used
//	// for property writes.
//	Operation string `json:"operation"`
//	// ThingID, digital twin thing ID
//	ThingID string `json:"thingID"`
//	// The name of the thing action or property (write)
//	Name string `json:"name"`
//	// The correlation ID to send the response to (reply-to)
//	RequestID string `json:"requestID"`
//	// the sender's connection ID to reply to. Internal use only.
//	ReplyTo string `json:"-"`
//	// Sender that invoked the action
//	SenderID string `json:"senderID"`
//	// Time of the last update
//	TimeUpdated string `json:"timeUpdated,omitempty"`
//}
