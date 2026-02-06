// Package messaging with the 3 flow messages: requests, response and  notifications
package messaging

import (
	"errors"
)

// Request status provided with the response.
// this aligns with action status values from WoT spec
const (
	// StatusPending - the request has not yet been delivered
	StatusPending = "pending"
	// StatusRunning - the request is being processed
	StatusRunning = "running"
	// StatusCompleted - the request processing was completed
	StatusCompleted = "completed"
	// StatusFailed - the request processing or delivery failed
	StatusFailed = "failed"
)

// ActionStatus is used for tracking the status of an action.
// NOTE: keep this in sync with the digital twin ActionStatus in the TD.
type ActionStatus struct {

	// ActionID that uniquely identifies the action instance.
	// This can be an identifier or a URL.
	ActionID string `json:"actionID,omitempty"`

	// Error with info when action failed
	Error *ErrorValue `json:"error,omitempty"`

	// Input of action
	Input any `json:"input,omitempty"`

	// Name of action
	Name string `json:"name,omitempty"`

	// Output with Action output
	Output any `json:"output,omitempty"`

	// SenderID of the client requesting the action
	SenderID string `json:"senderID,omitempty"`

	// State of action with progress: pending, running, completed, failed
	State string `json:"state,omitempty"`

	// ThingID digital-twin ThingID the action applies to
	ThingID string `json:"thingID,omitempty"`

	// Requested time the action request was received
	TimeRequested string `json:"timeRequested,omitempty"`

	// Updated time the action status was last updated
	TimeUpdated string `json:"timeUpdated,omitempty"`
}

// Error response payload
type ErrorValue struct {
	// Status code: https://w3c.github.io/wot-profile/#error-responses
	Status int `json:"status"`
	// Type is a URI reference [RFC3986] that identifies the problem type.
	Type string `json:"type"`
	// Title contains a short, human-readable summary of the problem type
	Title string `json:"title"`
	// Detail a human-readable explanation
	Detail string `json:"detail,omitempty"`
}

func (e *ErrorValue) String() string {
	return e.Title
}

// AsError returns an error instance or nil if no error is contained
func (e *ErrorValue) AsError() error {
	if e.Title == "" && e.Status == 0 {
		return nil
	}
	return errors.New(e.String())
}

// Create an ErrorValue object from the given error. This returns nil if err is nil.
func ErrorValueFromError(err error) *ErrorValue {
	if err == nil {
		return nil
	}
	return &ErrorValue{
		Status: 400, // bad request
		Type:   "Bad request",
		Title:  err.Error(),
		// Detail: "",
	}
}
