package dthing

import (
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
)

// DigitwinThing This is the digital twin Thing
// This contains the digital twin TD and the TD of the device it represents.
// It handles incoming requests and is a consumer of the actual device.
//
// In development. This will take over from the router.
type DigitwinThing struct {
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
}

// Handle request (from any client)
// - invoke action
// - write property
// - read property
// - read event
// - read action status

// Handle response (from device)
// - receive property update
// - receive event
// - receive action status
