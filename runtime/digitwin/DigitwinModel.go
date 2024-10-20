package digitwin

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/wot/tdd"
)

// Action and write-property progress status
//const (
//	StatusPending   = "pending"   // by service
//	StatusApplied   = "applied"   // by agent
//	StatusCompleted = "completed" // by thing
//	StatusFailed    = "failed"
//)

//type DigitalTwinActionValue struct {
//	// Action Progress: StatusPending, StatusApplied, StatusCompleted, StatusFailed
//	Progress string `json:"status"`
//
//	// Last updated action status in RFC 3339milli
//	Updated string `json:"updated"`
//
//	// Consumer invoking the action
//	SenderID string `json:"senderID"`
//
//	// Input data from consumer as per TD
//	Input any `json:"input,omitempty"`
//
//	// Output from agent as per TD. Set after completion.
//	// This is nil if no the action has no output data.
//	// Note that corresponding property value can carry the result of the action.
//	Output any `json:"output,omitempty"`
//
//	// MessageID of the action
//	MessageID string `json:"message-id,omitempty"`
//}

//type DigitalTwinEventValue struct {
//	// Event data from agent
//	Data any `json:"input,omitempty"`
//
//	// Time event value was updated
//	Updated string `json:"updated"`
//
//	// MessageID provided by the agent if the event is in response to something
//	// that carried a message id
//	MessageID string `json:"message-id,omitempty"`
//}

//type DigitalTwinPropertyValue struct {
//	// Data in the native format as provided by the agent and described in the
//	// TD property affordance.
//	Data any `json:"data,omitempty"`
//
//	// Timestamp the property value was updated by the agent, using RFC3339milli
//	Updated string `json:"updated,omitempty"`
//
//	// WriteSenderID is the account ID of the consumer, writing the property value
//	// to the hub.
//	// This is required and used in authorization of the sender and routing of messages.
//	// The underlying protocol binding MUST set this to the authenticated client.
//	WriteSenderID string `json:"senderID" `
//
//	// Requested property value as per TD
//	WriteData any `json:"input,omitempty"`
//
//	WriteUpdated string `json:"writeUpdated,omitempty"`
//
//	// WriteMessageID of the request
//	WriteMessageID string `json:"write-message-id"`
//
//	// WriteStatus: received, delivered, applied, completed, rejected, aborted
//	WriteStatus string `json:"status"`
//
//	// MessageID provided by the agent if this is a response to an action or write
//	MessageID string `json:"message-id,omitempty"`
//}

// DigitalTwinInstance contains the description and values of a digital twin Thing
type DigitalTwinInstance struct {
	// Agent that manages access to the exposed thing
	AgentID string `json:"agentID" `
	ID      string `json:"id" `

	// ThingTD as exposed by the agent
	ThingTD *tdd.TD `json:"agTD"`
	// DtwTD as exposed by the hub
	DtwTD *tdd.TD `json:"dtwTD" `

	// Latest properties as received from the exposed Thing
	PropValues map[string]digitwin.ThingValue `json:"pv"`
	// Latest events as received from the exposed Thing
	EventValues map[string]digitwin.ThingValue `json:"ev"`
	// Latest 'unsafe' actions as requested with their status
	ActionValues map[string]digitwin.ActionValue `json:"av"`
}
