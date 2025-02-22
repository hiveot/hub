package hiveotsseserver

// message types defined in the http basic profile
// https://w3c.github.io/wot-profile/#http-basic-profile-protocol-binding

// status values used in the HttpActionStatus response
const (
	HttpActionStatusPending   = "pending"
	HttpActionStatusRunning   = "running"
	HttpActionStatusCompleted = "completed"
	HttpActionStatusFailed    = "failed"
)

// HttpActionStatusMessage the status of an asynchronous action invocation request
type HttpActionStatusMessage struct {
	// Status is one of "pending","running","completed" or "failed"
	Status string `json:"status,omitempty"`

	Output any `json:"output,omitempty"`

	//Error message, if any, containing problem details format (RF7807)
	// For now just text
	Error string `json:"error,omitempty"`

	// The [URL] of an HttpActionStatus resource which can be used by queryaction
	// and cancelaction operations
	Href string `json:"href,omitempty"`

	// A timestamp indicating the time at which the Thing received the request
	// to execute the action.
	TimeRequested string `json:"timeRequested,omitempty"`

	//A timestamp indicating the time at which the Thing successfully completed
	//executing the action, or failed to execute the action.
	TimeEnded string `json:"timeEnded,omitempty"`
}

// https://w3c.github.io/wot-profile/#error-responses
// If an HTTP error response contains a body, the content of that body MUST conform
// with the Problem Details format [RFC7807].
// https://www.rfc-editor.org/rfc/rfc7807#section-3
type HttpErrorResponse struct {
	// URI reference identifying the problem type
	// Not sure what to put in here. Problem types don't exist in URI format.
	Type string `json:"type,omitempty"`
	// Human readable short summary
	Title string `json:"title"`
	// HTTP status code generated by the origin server
	// Likely 403 (no auth
	Status   int    `json:"status.omitempty"`
	Detail   string `json:"detail,omitempty"`   // Detail of the error
	Instance string `json:"instance,omitempty"` // Resource that identifies the occurrance of the problem
}
