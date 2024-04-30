// Package api with the JSON messaging API for the DigiTwin services
package api

import "github.com/hiveot/hub/lib/things"

// DigiTwinThingID digitwin thing identifier for addressing using things
const DigiTwinThingID = "digitwin"

// TODO1: define these methods through TD actions including their data schema
// TODO2: generate golang, js and python apis from the TD

// DigiTwinAgentID agent identifier for authentication
// (not used as this service is the hearth of the runtime.)
//const DigiTwinAgentID = "digitwin"

// DigiTwin service method keys
const (
	// Thing directory actions
	ReadThingMethod   = "readThing"
	ReadThingsMethod  = "readThings"
	RemoveThingMethod = "removeThing"
	//QueryThingMethod  = "queryThing"    // todo

	// Thing value actions
	ReadActionsMethod    = "readActions"
	ReadEventsMethod     = "readEvents"
	ReadPropertiesMethod = "readProperties"

	// Thing history actions
	ReadActionHistoryMethod = "readActionHistory"
	ReadEventHistoryMethod  = "readEventHistory"

	// WriteActionMethod requests to initiate an action on a Thing
	WriteActionMethod = "writeAction"
	// WritePropertyMethod requests to update a property configuration value
	WritePropertyMethod = "writeProperty"
)

// Action status values
const (
	// ActionStatusPending action is pending delivery
	ActionStatusPending = "pending"
	// ActionStatusDelivered action was delivered to the Thing
	ActionStatusDelivered = "delivered"
	// ActionStatusRejected action was rejected
	ActionStatusRejected = "rejected"
)

// ReadActionsArgs message parameters for the ReadActions method
// This returns the latest action values of a digital twin.
type ReadActionsArgs struct {
	// ThingID whose actions to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of action keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return action values that were sent since this ISO date (optional)
	Since string `json:"since,omitempty"` // YYYY-MM-DDTHH:MM:SS.sssTZ
}

// ReadActionsResp response contains the action messages that were requested
type ReadActionsResp struct {
	Messages things.ThingMessageMap `json:"messages,omitempty"`
}

// ReadActionHistoryArgs message parameters for the ReadActionHistory method
// This returns the historical values of an action.
type ReadActionHistoryArgs struct {
	// ThingID whose actions to get
	ThingID string `json:"thingID"`
	// Key of action to get
	Key string `json:"key,omitempty"`
	// Start date/time is ISO format
	Start string `json:"start,omitempty"` // YYYY-MM-DDTHH:MM:SS.sssTZ
	// End date/time is ISO format
	End string `json:"end,omitempty"` // YYYY-MM-DDTHH:MM:SS.sssTZ
}

// ReadActionHistoryResp response contains the action messages that were requested
type ReadActionHistoryResp struct {
	// Messages is the list of actions in order of date/time
	Messages []*things.ThingMessage `json:"messages,omitempty"`
	// ItemsRemaining flag, output was truncated. Repeat the request with the last timestamp
	ItemsRemaining bool `json:"itemsRemaining"`
}

// ReadEventsArgs message parameters for the ReadEvents method
type ReadEventsArgs struct {
	// ThingID whose values to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of event keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return values that were sent since this ISO date (optional)
	Since string `json:"since,omitempty"`
}

// ReadEventsResp response
type ReadEventsResp struct {
	Messages things.ThingMessageMap `json:"messages,omitempty"`
}

// ReadEventHistoryArgs message parameters for the ReadEventHistory method
// This returns the historical values of an event.
type ReadEventHistoryArgs struct {
	// ThingID whose actions to get
	ThingID string `json:"thingID"`
	// Key of action to get
	Key string `json:"key,omitempty"`
	// Start date/time is ISO format
	Start string `json:"start,omitempty"` // YYYY-MM-DDTHH:MM:SS.sssTZ
	// End date/time is ISO format
	End string `json:"end,omitempty"` // YYYY-MM-DDTHH:MM:SS.sssTZ
}

// ReadEventHistoryResp response contains the event messages that were requested
type ReadEventHistoryResp struct {
	// Messages is the list of events in order of date/time
	Messages []*things.ThingMessage `json:"messages,omitempty"`
	// ItemsRemaining flag, output was truncated. Repeat the request with the last timestamp
	ItemsRemaining bool `json:"itemsRemaining"`
}

// ReadPropertiesArgs message parameters for the ReadProperties method
type ReadPropertiesArgs struct {
	// ThingID whose values to get
	ThingID string `json:"thingID"`
	// Keys is an optional filter list of property keys to return
	Keys []string `json:"keys,omitempty"`
	// Since only return values that were sent since this ISO date (optional)
	Since string `json:"since,omitempty"`
}

// ReadPropertiesResp response
type ReadPropertiesResp struct {
	Messages things.ThingMessageMap `json:"messages,omitempty"`
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

// WriteActionArgs arguments to request an action from a thing
type WriteActionArgs struct {
	ThingID string `json:"thingID"`
	Key     string `json:"key"`
	Value   []byte `json:"value"` // json encoded
}

// WriteActionResp returns the raw action response as defined in the Thing's TD
type WriteActionResp struct {
	// Status TODO: is the action status, pending, completed, error, rejected
	ActionID string `json:"actionID"`
	Status   string `json:"status"`
	Value    []byte `json:"value" `
}

// WriteEventArgs arguments to publish a single event
type WriteEventArgs struct {
	ThingID string `json:"thingID"`
	Key     string `json:"key"`
	Value   []byte `json:"value"` // json encoded
}

// WritePropertyArgs arguments to request a property change from a thing
type WritePropertyArgs struct {
	ThingID string `json:"thingID"`
	Key     string `json:"key"`
	Value   string `json:"value"` // stringified value
}

type WriteThingArgs struct {
	TD *things.TD `json:"td"` // json encoded td
}
