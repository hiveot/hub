package things

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/utils"
	"sync"
	"time"
)

const WoTTDContext = "https://www.w3.org/2022/wot/td/v1.1"
const HiveOTContext = "https://www.hiveot.net/vocab/v0.1"

// TD contains the Thing Description document
// Its structure is:
//
//	{
//	     @context: <WoTTDContext>, {"ht":<HiveOTContext>},
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
	// in order to add the "ht" namespace, the context value can be a string or map
	AtContext []any `json:"@context"`

	// JSON-LD keyword to label the object with semantic tags (or types).
	// in HiveOT this contains the device type defined in the vocabulary.
	// Intended for grouping and querying similar devices, and standardized presentation such as icons
	AtType  string `json:"@type,omitempty"`
	AtTypes string `json:"@types,omitempty"`

	// base: Define the base URI that is used for all relative URI references throughout a TD document.
	//Base string `json:"base,omitempty"`

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
	// * IDs are optional. However, in HiveOT that won't work as they must be addressable.
	// * IDs start with "urn:" based on the idea that IDs can be used as an address. In HiveOT, IDs and Addresses
	//   serve a different purpose. IDs are not addresses. HiveOT allows IDs that do not start with "urn:".
	//   note that pubsub uses addresses of which the things ID is part of.
	// * ID's SHOULD be mutable. Recommended is on device reset the ID is changed.
	// * The id of a TD SHOULD NOT contain metadata describing the Thing or from the TD itself.
	// * Using random UUIDs as recommended in 10.5
	ID string `json:"id,omitempty"`

	// Information about the TD maintainer as URI scheme (e.g., mailto [RFC6068], tel [RFC3966], https).
	Support string `json:"support,omitempty"`

	// Title is a short description of the thing in human-readable in the default language. Required.
	Title string `json:"title"`
	// Human-readable titles in the different languages
	Titles map[string]string `json:"titles,omitempty"`

	// All properties-based interaction affordances of the things
	Properties map[string]*PropertyAffordance `json:"properties,omitempty"`
	// All action-based interaction affordances of the things
	Actions map[string]*ActionAffordance `json:"actions,omitempty"`
	// All event-based interaction affordances of the things
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
//	name is the action name under which it is stored in the action affordance map.
//	actionType from the vocabulary or "" if this is a non-standardized action
//	title is the short display title of the action
//	description optional explanation of the action
//	input with dataschema of the action input data, if any
func (tdoc *TD) AddAction(name string, actionType string, title string, description string, input *DataSchema) *ActionAffordance {
	actionAff := &ActionAffordance{
		ActionType:  actionType,
		Title:       title,
		Description: description,
		Input:       input,
	}
	tdoc.UpdateAction(name, actionAff)
	return actionAff
}

// AddDimmerAction is short for adding an action to control a dimmer
// This includes a data schema for integers
//
//	actionID is the instance ID of the action, unique within the Thing
func (tdoc *TD) AddDimmerAction(actionID string) *ActionAffordance {
	act := tdoc.AddAction(actionID, vocab.ActionDimmer, "", "", &DataSchema{
		AtType: vocab.ActionDimmer,
		Type:   vocab.WoTDataTypeInteger,
	})
	return act
}

// AddDimmerEvent is short for adding an event for a dimmer change event
// This includes a data schema for integers
//
//	eventID is the instance ID of the event, unique within the Thing
func (tdoc *TD) AddDimmerEvent(eventID string) *EventAffordance {
	ev := tdoc.AddEvent(eventID, vocab.PropSwitchDimmer, "", "", &DataSchema{
		AtType: vocab.PropSwitchDimmer,
		Type:   vocab.WoTDataTypeInteger,
	})
	return ev
}

// AddEvent provides a simple way to add an event to the TD.
// This returns the event affordance that can be augmented/modified directly.
//
// If the event returns data then set the .Data field to a DataSchema instance that describes it.
//
//	name is the event name under which it is stored in the affordance map.
//	eventType describes the type of event in HiveOT vocabulary if available, or "" if non-standard.
//	title is the short display title of the event
//	schema optional event data schema or nil if the event doesn't carry any data
func (tdoc *TD) AddEvent(
	name string, eventType string, title string, description string, schema *DataSchema) *EventAffordance {

	evAff := &EventAffordance{
		EventType:   eventType,
		Title:       title,
		Description: description,
		Data:        schema,
	}

	tdoc.UpdateEvent(name, evAff)
	return evAff
}

// AddProperty provides a simple way to add a read-only property to the TD.
//
// This returns the property affordance that can be augmented/modified directly
// By default the property is a read-only attribute.
//
//	name is the property name under which it is stored in the affordance map.
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
//	title is the short display title of the property.
//	dataType is the type of data the property holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *TD) AddProperty(name string, propType string, title string, dataType string) *PropertyAffordance {
	prop := &PropertyAffordance{
		DataSchema: DataSchema{
			AtType:   propType,
			Title:    title,
			Type:     dataType,
			ReadOnly: true,
			//InitialValue: initialValue,
		},
	}
	tdoc.UpdateProperty(name, prop)
	return prop
}

// AddPropertyAsString is short for adding a read-only string property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsString(name string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(name, propType, title, vocab.WoTDataTypeString)
}

// AddPropertyAsBool is short for adding a read-only boolean property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsBool(name string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(name, propType, title, vocab.WoTDataTypeBool)
}

// AddPropertyAsInt is short for adding a read-only integer property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsInt(name string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(name, propType, title, vocab.WoTDataTypeInteger)
}

// AddSwitchAction is short for adding an action to control an on/off switch
func (tdoc *TD) AddSwitchAction(actionID string) *ActionAffordance {
	act := tdoc.AddAction(actionID, vocab.ActionSwitchOnOff, "", "",
		&DataSchema{
			AtType: vocab.ActionSwitchOnOff,
			Type:   vocab.WoTDataTypeBool,
			Enum:   []interface{}{"on", "off"},
		})
	return act
}

// AddSwitchEvent is short for adding an event for a switch
func (tdoc *TD) AddSwitchEvent(eventID string) *EventAffordance {
	ev := tdoc.AddEvent(eventID, vocab.PropSwitchOnOff, "", "",
		&DataSchema{
			AtType: vocab.PropSwitchOnOff,
			Type:   vocab.WoTDataTypeBool,
			Enum:   []interface{}{"on", "off"},
		})
	return ev
}

// AddSensorEvent is short for adding an event for a generic sensor
func (tdoc *TD) AddSensorEvent(eventID string) *EventAffordance {
	ev := tdoc.AddEvent(eventID, vocab.PropEnv, "", "",
		&DataSchema{
			AtType: vocab.PropEnv,
			Type:   vocab.WoTDataTypeNumber,
		})
	return ev
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
// Returns nil if name is not an action or no affordance is defined.
func (tdoc *TD) GetAction(name string) *ActionAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	actionAffordance, found := tdoc.Actions[name]
	if !found {
		return nil
	}
	return actionAffordance
}

// GetAge returns the age of the document since last modified in a human readable format.
// this is just an experiment to see if this is useful.
// Might be better to do on the UI client side to reduce cpu.
func (tdoc *TD) GetAge() string {
	t, err := dateparse.ParseAny(tdoc.Modified)
	if err != nil {
		return tdoc.Modified
	}
	return utils.Age(t)
}

// GetEvent returns the Schema for the event or nil if the event doesn't exist
func (tdoc *TD) GetEvent(name string) *EventAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	eventAffordance, found := tdoc.Events[name]
	if !found {
		return nil
	}
	return eventAffordance
}

// GetProperty returns the Schema and value for the property or nil if its key is not found.
func (tdoc *TD) GetProperty(key string) *PropertyAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()
	propAffordance, found := tdoc.Properties[key]
	if !found {
		return nil
	}
	return propAffordance
}

// GetPropertyOfType returns the first property affordance with the given @type
// This returns the property ID and the property affordances, or nil if not found
func (tdoc *TD) GetPropertyOfType(atType string) (string, *PropertyAffordance) {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()
	for propID, prop := range tdoc.Properties {
		if prop.AtType == atType {
			return propID, prop
		}
	}
	return "", nil
}

// GetID returns the ID of the things TD
func (tdoc *TD) GetID() string {
	return tdoc.ID
}

// UpdateAction adds a new or replaces an existing action affordance of actionID. Intended for creating TDs.
//
//	name is the name under which to store the action affordance.
//
// This returns the added action affordance
func (tdoc *TD) UpdateAction(name string, affordance *ActionAffordance) *ActionAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Actions[name] = affordance
	return affordance
}

// UpdateEvent adds a new or replaces an existing event affordance of eventID. Intended for creating TDs
//
//	name is the ID under which to store the event affordance.
//
// This returns the added event affordance.
func (tdoc *TD) UpdateEvent(name string, affordance *EventAffordance) *EventAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Events[name] = affordance
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
//	name is the name under which to store the property affordance.
//
// This returns the added affordance to support chaining
func (tdoc *TD) UpdateProperty(name string, affordance *PropertyAffordance) *PropertyAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Properties[name] = affordance
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
//	     @context: "http://www.w3.org/ns/td",{"ht":"http://hiveot.net/vocab/v..."}
//	     @type: <deviceType>,        // required in HiveOT. See DeviceType vocabulary
//	     id: <thingID>,              // urn:[{prefix}:]{randomID}   required in hiveot
//	     title: string,              // required. Human description of the things
//	     created: <iso8601>,         // will be the current timestamp. See vocabulary TimeFormat
//	     actions: {name:TDAction, ...},
//	     events:  {name: TDEvent, ...},
//	     properties: {name: TDProperty, ...}
//	}
func NewTD(thingID string, title string, deviceType string) *TD {
	td := TD{
		AtContext: []any{
			WoTTDContext,
			map[string]string{"ht": HiveOTContext},
		},
		AtType:     deviceType,
		Actions:    map[string]*ActionAffordance{},
		Created:    time.Now().Format(utils.ISO8601Format),
		Events:     map[string]*EventAffordance{},
		Forms:      nil,
		ID:         thingID,
		Modified:   time.Now().Format(utils.ISO8601Format),
		Properties: map[string]*PropertyAffordance{},
		// security schemas don't apply to HiveOT devices, except services exposed by the hub itself
		Security:    vocab.WoTNoSecurityScheme,
		Title:       title,
		updateMutex: sync.RWMutex{},
	}

	return &td
}
