package store

import (
	"github.com/hiveot/gocore/wot/td"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
)

type ValueQueue struct {
	values   []digitwin.ThingValue
	maxDepth int
}

// DigitalTwinInstance contains the digital twin of a device
type DigitalTwinInstance struct {
	// Agent that manages access to the exposed thing
	AgentID string `json:"agentID" `
	ID      string `json:"id" `

	// AgentTD as exposed by the agent
	AgentTD *td.TD `json:"agTD"`
	// DigitwinTD as exposed by the hub
	DigitwinTD *td.TD `json:"dtwTD" `

	// PropValues with latest digital twin properties
	PropValues map[string]digitwin.ThingValue `json:"pv"`
	// EventValues with latest digital twin events
	EventValues map[string]digitwin.ThingValue `json:"ev"`
	// ActionStatuses with latest 'unsafe' digital twin action status
	ActionStatuses map[string]digitwin.ActionStatus `json:"av"`

	// TBD: queue actions in the inbox of this device for timed delivery
	//Inbox ActionQueue `json:"inbox"`
	// TBD: queue events in the outbox to allow reading recent events on connecting
	//Outbox ValueQueue `json:"outbox"`
}
