package transports

// request status
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
