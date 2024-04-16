package api

import "github.com/hiveot/hub/lib/things"

// DigiTwinServiceID digitwin service identifier
const DigiTwinServiceID = "digitwin"

// Methods for accessing Thing description documents
const (
	ReadThingMethod   = "readThing"
	ReadThingsMethod  = "readThings"
	RemoveThingMethod = "removeThing"
)

// Methods for accessing Thing values
const (
	ReadActionsMethod    = "getActions"
	ReadEventsMethod     = "getEvents"
	ReadPropertiesMethod = "getProperties"
)

// ReadActionsArgs message parameters for the ReadActions method
// This returns the latest action values of a digital twin.
type ReadActionsArgs struct {
	// ThingID whose actions to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of action keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return action values that were sent since this ISO date (optional)
	//Since string `json:"since,omitempty"`
}

// ReadActionsResp response
type ReadActionsResp struct {
	Actions things.ThingMessageMap `json:"actions,omitempty"`
}

// ReadEventsArgs message parameters for the ReadEvents method
type ReadEventsArgs struct {
	// ThingID whose values to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of event keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return values that were sent since this ISO date (optional)
	//Since string `json:"since,omitempty"`
}

// ReadEventsResp response
type ReadEventsResp struct {
	Events things.ThingMessageMap `json:"events,omitempty"`
}

// ReadPropertiesArgs message parameters for the ReadProperties method
type ReadPropertiesArgs struct {
	// ThingID whose values to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of property keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return values that were sent since this ISO date (optional)
	//Since string `json:"since,omitempty"`
}

// ReadPropertiesResp response
type ReadPropertiesResp struct {
	Props things.ThingMessageMap `json:"props,omitempty"`
}

// ReadThingArgs arguments to read a single Thing TD document
type ReadThingArgs struct {
	ThingID string `json:"thingID"`
}

// ReadThingResp response containing a single TD document
type ReadThingResp struct {
	TD *things.TD `json:"td"`
}

// ReadThingsArgs arguments to read multiple Thing TD documents
type ReadThingsArgs struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

// ReadThingsResp response containing a list of TD documents
type ReadThingsResp struct {
	TDs []*things.TD `json:"tds"`
}

// RemoveThingArgs arguments for removing a TD document
type RemoveThingArgs struct {
	ThingID string `json:"thingID"`
}
