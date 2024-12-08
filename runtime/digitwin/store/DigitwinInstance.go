package store

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/wot/td"
)

type ActionQueue struct {
	actions []digitwin.ActionValue
}
type ValueQueue struct {
	values   []digitwin.ThingValue
	maxDepth int
}

// DigitalTwinInstance contains the digital twin of a device
type DigitalTwinInstance struct {
	// Agent that manages access to the exposed thing
	AgentID string `json:"agentID" `
	ID      string `json:"id" `

	// ThingTD as exposed by the agent
	ThingTD *td.TD `json:"agTD"`
	// DtwTD as exposed by the hub
	DtwTD *td.TD `json:"dtwTD" `

	// Latest properties as received from the exposed Thing
	PropValues map[string]digitwin.ThingValue `json:"pv"`
	// Latest events as received from the exposed Thing
	EventValues map[string]digitwin.ThingValue `json:"ev"`
	// Latest 'unsafe' actions as requested with their status
	ActionValues map[string]digitwin.ActionValue `json:"av"`

	// TBD: queue actions in the inbox of this device for timed delivery
	//Inbox ActionQueue `json:"inbox"`
	// TBD: queue events in the outbox to allow reading recent events on connecting
	//Outbox ValueQueue `json:"outbox"`
}
