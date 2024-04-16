package api

import "github.com/hiveot/hub/lib/things"

// ValueServiceID Directory service 'thingID'
const ValueServiceID = "valueService"

// Service crud method keys
const (
	ValueServiceGetActionsMethod    = "getActions"
	ValueServiceGetPropertiesMethod = "getProperties"
	ValueServiceGetEventsMethod     = "getEvents"
)

// GetActionsArgs message parameters for the GetActions method
// This returns the latest action values.
type GetActionsArgs struct {
	// ThingID whose actions to get
	ThingID string `json:"thingID"`
	// Keys specifies a list of action keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return action values that were sent since this date (optional)
	Since string `json:"since,omitempty"`
}

// GetPropertiesArgs message parameters for the GetProperties method
type GetPropertiesArgs struct {
	// ThingID whose properties to get
	ThingID string `json:"thingID"`
	// Keys specifies a list of property names to return
	Keys []string `json:"keys,omitempty"`
	// Since only return action values that were sent since this date (optional)
	Since string `json:"since,omitempty"`
}

// GetPropertiesResp response
type GetPropertiesResp struct {
	Props things.ThingMessageMap `json:"props,omitempty"`
}
