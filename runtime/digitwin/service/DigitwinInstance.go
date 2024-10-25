package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/wot/tdd"
)

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
