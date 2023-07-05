package thing

import (
	vocab2 "github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"sync"
	"time"
)

// TD contains the Thing Description document
// Its structure is:
//
//	{
//	     @context: "http://www.w3.org/ns/td",
//	     @type: <deviceType>,
//	     id: <thingID>,
//	     title: <human description>,  (why is this not a property?)
//	     modified: <iso8601>,
//	     actions: {actionID: ActionAffordance, ...},
//	     events:  {eventID: EventAffordance, ...},
//	     properties: {propID: PropertyAffordance, ...}
//	}
type TD struct {
	// JSON-LD keyword to define shorthand names called terms that are used throughout a TD document. Required.
	AtContext []string `json:"@context"`

	// JSON-LD keyword to label the object with semantic tags (or types).
	// in HiveOT this contains the device type defined in the vocabulary.
	// Intended for grouping and querying similar devices, and standardized presentation such as icons
	DeviceType string `json:"@type,omitempty"`
	AtTypes    string `json:"@types,omitempty"`

	// base: Define the base URI that is used for all relative URI references throughout a TD document.
	Base string `json:"base,omitempty"`

	// ISO8601 timestamp this document was first created
	Created string `json:"created,omitempty"`
	// ISO8601 timestamp this document was last modified
	Modified string `json:"modified,omitempty"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Version information of the TD document (?not the device??)
	//Version VersionInfo `json:"version,omitempty"` // todo

	// Instance identifier of the Thing in form of a URI (RFC3986)
	// https://www.w3.org/TR/wot-thing-description11/#sec-privacy-consideration-id
	// * IDs are optional. However in HiveOT that won't work as they must be addressable.
	// * IDs start with "urn:" based on the idea that IDs can be used as an address. In HiveOT, IDs and Addresses
	//   serve a different purpose. IDs are not addresses. HiveOT allows IDs that do not start with "urn:".
	//   note that pubsub uses addresses of which the thing ID is part of.
	// * ID's SHOULD be mutable. Recommended is on device reset the ID is changed.
	// * The id of a TD SHOULD NOT contain metadata describing the Thing or from the TD itself.
	// * Using random UUIDs as recommended in 10.5
	ID string `json:"id,omitempty"`

	// Information about the TD maintainer as URI scheme (e.g., mailto [RFC6068], tel [RFC3966], https).
	Support string `json:"support,omitempty"`

	// Human-readable title in the default language. Required.
	Title string `json:"title"`
	// Human-readable titles in the different languages
	Titles map[string]string `json:"titles,omitempty"`

	// All properties-based interaction affordances of the thing
	Properties map[string]*PropertyAffordance `json:"properties,omitempty"`
	// All action-based interaction affordances of the thing
	Actions map[string]*ActionAffordance `json:"actions,omitempty"`
	// All event-based interaction affordances of the thing
	Events map[string]*EventAffordance `json:"events,omitempty"`

	// links: todo

	// Form hypermedia controls to describe how an operation can be performed. Forms are serializations of
	// Protocol Bindings. Thing-level forms are used to describe endpoints for a group of interaction affordances.
	Forms []Form `json:"forms,omitempty"`

	// Set of security definition names, chosen from those defined in securityDefinitions
	// In HiveOT security is handled by the Hub. HiveOT Things will use the NoSecurityScheme type
	Security string `json:"security"`
	// Set of named security configurations (definitions only).
	// Not actually applied unless names are used in a security name-value pair. (why is this mandatory then?)
	SecurityDefinitions map[string]string `json:"securityDefinitions,omitempty"`

	// profile: todo
	// schemaDefinitions: todo
	// uriVariables: todo
	updateMutex sync.RWMutex
}

// AddAction provides a simple way to add an action affordance Schema to the TD.
// This returns the action affordance that can be augmented/modified directly.
//
// If the action accepts input parameters then set the .Data field to a DataSchema instance that
// describes the parameter(s).
//
//	id is the action instance ID under which it is stored in the action affordance map.
//	actionType from the vocabulary or "" if this is a non-standardized action
//	title is the short display title of the action
//	description optional explanation of the action
//	input with dataschema of the action input data, if any
func (tdoc *TD) AddAction(id string, actionType string, title string, description string, input *DataSchema) *ActionAffordance {
	actionAff := &ActionAffordance{
		ActionType:  actionType,
		Title:       title,
		Description: description,
		Input:       input,
	}
	tdoc.UpdateAction(id, actionAff)
	return actionAff
}

// AddEvent provides a simple way to add an event to the TD.
// This returns the event affordance that can be augmented/modified directly.
//
// If the event returns data then set the .Data field to a DataSchema instance that describes it.
//
//	id is the unique event instance ID under which it is stored in the affordance map.
//	eventType describes the type of event in HiveOT vocabulary if available, or "" if non-standard.
//	title is the short display title of the event
//	dataType is the type of data the event holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
//	schema optional event data schema or nil if the event doesn't carry any data
func (tdoc *TD) AddEvent(id string, eventType string, title string, description string, schema *DataSchema) *EventAffordance {
	evAff := &EventAffordance{
		EventType:   eventType,
		Title:       title,
		Description: description,
		Data:        schema,
	}
	tdoc.UpdateEvent(id, evAff)
	return evAff
}

// AddProperty provides a simple way to add a property to the TD
// This returns the property affordance that can be augmented/modified directly
// By default the property is a read-only attribute.
//
//	id is the unique property instance ID under which it is stored in the affordance map.
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
//	title is the short display title of the property.
//	dataType is the type of data the property holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
//	initialValue optional value, in text format, at time of creation. Intended for testing or debugging
func (tdoc *TD) AddProperty(id string, propType string, title string, dataType string, initialValue string) *PropertyAffordance {
	prop := &PropertyAffordance{
		DataSchema: DataSchema{
			AtType:       propType,
			Title:        title,
			Type:         dataType,
			ReadOnly:     true,
			InitialValue: initialValue,
		},
	}
	tdoc.UpdateProperty(id, prop)
	return prop
}

// AsMap returns the TD document as a map
func (tdoc *TD) AsMap() map[string]interface{} {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	var asMap map[string]interface{}
	asJSON, _ := ser.Marshal(tdoc)
	_ = ser.Unmarshal(asJSON, &asMap)
	return asMap
}

// tbd json-ld parsers:
// Most popular; https://github.com/xeipuuv/gojsonschema
// Other:  https://github.com/piprate/json-gold

// GetAction returns the action affordance with Schema for the action.
// Returns nil if actionID is not an action or no affordance is defined.
func (tdoc *TD) GetAction(actionID string) *ActionAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	actionAffordance, found := tdoc.Actions[actionID]
	if !found {
		return nil
	}
	return actionAffordance
}

// GetEvent returns the Schema for the event or nil if the event doesn't exist
func (tdoc *TD) GetEvent(eventID string) *EventAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	eventAffordance, found := tdoc.Events[eventID]
	if !found {
		return nil
	}
	return eventAffordance
}

// GetProperty returns the Schema and value for the property or nil if propID is not a property
func (tdoc *TD) GetProperty(propID string) *PropertyAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()
	propAffordance, found := tdoc.Properties[propID]
	if !found {
		return nil
	}
	return propAffordance
}

// GetID returns the ID of the thing TD
func (tdoc *TD) GetID() string {
	return tdoc.ID
}

// UpdateAction adds a new or replaces an existing action affordance of actionID. Intended for creating TDs.
//
//	actionID is the ID under which to store the affordance.
//
// This returns the added action affordance
func (tdoc *TD) UpdateAction(actionID string, affordance *ActionAffordance) *ActionAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Actions[actionID] = affordance
	return affordance
}

// UpdateEvent adds a new or replaces an existing event affordance of eventID. Intended for creating TDs
//
//	eventID is the ID under which to store the affordance.
//
// This returns the added event affordance.
func (tdoc *TD) UpdateEvent(eventID string, affordance *EventAffordance) *EventAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Events[eventID] = affordance
	return affordance
}

// UpdateForms sets the top level forms section of the TD
// NOTE: In HiveOT actions are always routed via the Hub using the Hub's protocol binding.
// Under normal circumstances forms are therefore not needed.
func (tdoc *TD) UpdateForms(formList []Form) {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Forms = formList
}

// UpdateProperty adds or replaces a property affordance in the TD. Intended for creating TDs
//
//	propID is the ID under which to store the affordance.
//
// This returns the added affordance to support chaining
func (tdoc *TD) UpdateProperty(propID string, affordance *PropertyAffordance) *PropertyAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Properties[propID] = affordance
	return affordance
}

// UpdateTitleDescription sets the title and description of the Thing in the default language
func (tdoc *TD) UpdateTitleDescription(title string, description string) {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Title = title
	tdoc.Description = description
}

// NewTD creates a new Thing Description document with properties, events and actions
// Its structure:
//
//	{
//	     @context: "http://www.w3.org/ns/td",
//	     id: <thingID>,              // urn:[{prefix}:]{randomID}
//	     title: string,              // required. Human description of the thing
//	     @type: <deviceType>,        // required in HiveOT. See DeviceType vocabulary
//	     created: <iso8601>,         // will be the current timestamp. See vocabulary TimeFormat
//	     actions: {actionID:TDAction, ...},
//	     events:  {eventID: TDEvent, ...},
//	     properties: {propID: TDProperty, ...}
//	}
func NewTD(thingID string, title string, deviceType string) *TD {
	td := TD{
		AtContext: []string{"http://www.w3.org/ns/thing"},
		// TODO @type is a JSON-LD keyword to label using semantic tags, eg it needs a Schema
		DeviceType: deviceType,
		Actions:    map[string]*ActionAffordance{},
		Created:    time.Now().Format(vocab2.ISO8601Format),
		Events:     map[string]*EventAffordance{},
		Forms:      nil,
		ID:         thingID,
		Modified:   time.Now().Format(vocab2.ISO8601Format),
		Properties: map[string]*PropertyAffordance{},
		// security schemas don't apply to HiveOT devices, except services exposed by the hub itself
		Security:    vocab2.WoTNoSecurityScheme,
		Title:       title,
		updateMutex: sync.RWMutex{},
	}

	return &td
}
