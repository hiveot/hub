package tdd

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/utils"
	"net/url"
	"strings"
	"sync"
	"time"
)

const WoTTDContext = "https://www.w3.org/2022/wot/td/v1.1"
const HiveOTContext = "https://www.hiveot.net/vocab/v0.1"

// TD contains the Thing Description document
// Its structure is:
//
//	{
//		@context: <WoTTDContext>, {"ht":<HiveOTContext>},
//		@type: <deviceType>,
//		id: <thingID>,
//		title: <human description>,  (why is this not a property?)
//		modified: <iso8601>,
//		schemaDefinitions: {...},
//		actions: {actionID: ActionAffordance, ...},
//		events:  {eventID: EventAffordance, ...},
//		properties: {propID: PropertyAffordance, ...}
//	 }
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

	// ISO8601 timestamp this document was first created. See also 'Modified'.
	Created string `json:"created,omitempty"`

	// Describe the device in the default human-readable language.
	// It is recommended to use the product description.
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Version information of the TD document (?not the device??)
	//Version VersionInfo `json:"version,omitempty"` // todo

	// ID is the Thing instance identifier.
	// HiveOT prefixes the thing's ID with the publishing agentID to help with uniqueness, separated by colon:
	//  ThingID format used: urn:agentID:deviceID
	ID string `json:"id,omitempty"`

	// ISO8601 timestamp this document was last modified. See also 'Created'.
	Modified string `json:"modified,omitempty"`

	// Base URI for all relative URI references
	Base string `json:"base,omitempty"`

	// Information about the TD maintainer as URI scheme (e.g., mailto [RFC6068], tel [RFC3966], https).
	Support string `json:"support,omitempty"`

	// Title is the name of the thing in human-readable in the default language. Required.
	// The same name is provided through the properties configuration, where it can be changed.
	// This allows to present the TD by its name without having to load its property values.
	Title string `json:"title"`
	// Human-readable titles in the different languages
	Titles map[string]string `json:"titles,omitempty"`

	SchemaDefinitions map[string]*DataSchema `json:"schemaDefinitions,omitempty"`

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

	// Security is a string or array of security definition names, chosen from those defined
	// in securityDefinitions.
	// In HiveOT security is handled by the Hub. HiveOT Things will use the NoSecurityScheme type
	//Security string `json:"security"`
	// FIXME: make WoT compliant and include the transport protocols authentication
	Security any `json:"security"`

	// Set of named security configurations (definitions only).
	// Not actually applied unless names are used in a security name-value pair. (why is this mandatory then?)
	// FIXME: make WoT compliant and include the transport protocols authentication
	SecurityDefinitions map[string]SecurityScheme `json:"securityDefinitions"`

	// for use to access writable data, when TD's are updated on the fly by agents.
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
//	eventName is the name under which it is stored in the affordance map.
//		If omitted, the eventType is used.
//	eventType describes the type of event in HiveOT vocabulary if available, or "" if non-standard.
//	title is the short display title of the event
//	schema optional event data schema or nil if the event doesn't carry any data
func (tdoc *TD) AddEvent(
	eventName string, eventType string, title string, description string, schema *DataSchema) *EventAffordance {
	if eventName == "" {
		eventName = eventType
	}
	evAff := &EventAffordance{
		EventType:   eventType,
		Title:       title,
		Description: description,
		//Data:        *schema,
	}
	if schema != nil {
		evAff.Data = *schema
	}

	tdoc.UpdateEvent(eventName, evAff)
	return evAff
}

// AddForms adds top level forms to the TD
// Existing forms are retained.
func (tdoc *TD) AddForms(forms []Form) {
	tdoc.Forms = append(tdoc.Forms, forms...)
}

// AddProperty provides a simple way to add a read-only property to the TD.
//
// This returns the property affordance that can be augmented/modified directly
// By default the property is a read-only attribute.
//
//	propName is the name under which it is stored in the affordance map.
//		If omitted, the eventType is used.
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
//	title is the short display title of the property.
//	dataType is the type of data the property holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *TD) AddProperty(propName string, propType string, title string, dataType string) *PropertyAffordance {
	if propName == "" {
		propName = propType
	}
	prop := &PropertyAffordance{
		DataSchema: DataSchema{
			AtType:   propType,
			Title:    title,
			Type:     dataType,
			ReadOnly: true,
			//InitialValue: initialValue,
		},
	}
	tdoc.UpdateProperty(propName, prop)
	return prop
}

// AddPropertyAsString is short for adding a read-only string property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsString(propName string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(propName, propType, title, vocab.WoTDataTypeString)
}

// AddPropertyAsBool is short for adding a read-only boolean property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsBool(propName string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(propName, propType, title, vocab.WoTDataTypeBool)
}

// AddPropertyAsInt is short for adding a read-only integer property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsInt(propName string, propType string, title string) *PropertyAffordance {
	return tdoc.AddProperty(propName, propType, title, vocab.WoTDataTypeInteger)
}

// AddSwitchAction is short for adding an action to control an on/off switch
func (tdoc *TD) AddSwitchAction(actionName string, title string) *ActionAffordance {
	act := tdoc.AddAction(actionName, vocab.ActionSwitchOnOff, title, "",
		&DataSchema{
			AtType: vocab.ActionSwitchOnOff,
			Type:   vocab.WoTDataTypeBool,
			Enum:   []interface{}{"on", "off"},
		})
	return act
}

// AddSwitchEvent is short for adding an event for a switch
func (tdoc *TD) AddSwitchEvent(eventName string, title string) *EventAffordance {
	ev := tdoc.AddEvent(eventName, vocab.PropSwitchOnOff, title, "",
		&DataSchema{
			AtType: vocab.PropSwitchOnOff,
			Type:   vocab.WoTDataTypeBool,
			Enum:   []interface{}{"on", "off"},
		})
	return ev
}

// AddSensorEvent is short for adding an event for a generic sensor
func (tdoc *TD) AddSensorEvent(eventName string, title string) *EventAffordance {
	ev := tdoc.AddEvent(eventName, vocab.PropEnv, title, "",
		&DataSchema{
			AtType: vocab.PropEnv,
			Type:   vocab.WoTDataTypeNumber,
		})
	return ev
}

// AsMap returns the TD document as a key-value map
func (tdoc *TD) AsMap() map[string]interface{} {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	var asMap map[string]interface{}
	asJSON, _ := ser.Marshal(tdoc)
	_ = ser.Unmarshal(asJSON, &asMap)
	return asMap
}

// EscapeKeys replace spaces in property, event and action keys with dashes
func (tdoc *TD) EscapeKeys() {
	// NOTE: this does not modify links and refs. Those must be valid.
	newProps := make(map[string]*PropertyAffordance)
	for k, v := range tdoc.Properties {
		kesc := strings.ReplaceAll(k, " ", "-")
		newProps[kesc] = v
	}
	tdoc.Properties = newProps

	newEvents := make(map[string]*EventAffordance)
	for k, v := range tdoc.Events {
		kesc := strings.ReplaceAll(k, " ", "-")
		newEvents[kesc] = v
	}
	tdoc.Events = newEvents

	newActions := make(map[string]*ActionAffordance)
	for k, v := range tdoc.Actions {
		kesc := strings.ReplaceAll(k, " ", "-")
		newActions[kesc] = v
	}
	tdoc.Actions = newActions
}

// tbd json-ld parsers:
// Most popular; https://github.com/xeipuuv/gojsonschema
// Other:  https://github.com/piprate/json-gold

// GetAction returns the action affordance with Schema for the action.
// Returns nil if name is not an action or no affordance is defined.
func (tdoc *TD) GetAction(actionName string) *ActionAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	actionAffordance, found := tdoc.Actions[actionName]
	if !found {
		return nil
	}
	return actionAffordance
}

//// GetAge returns the age of the document since last modified in a human-readable format.
//// this is just an experiment to see if this is useful.
//// Might be better to do on the UI client side to reduce cpu.
//func (tdoc *TD) GetAge() string {
//	t, err := dateparse.ParseAny(tdoc.Modified)
//	if err != nil {
//		return tdoc.Modified
//	}
//	return utils.Age(t)
//}

// GetAtTypeVocab return the vocab map of the @type
func (tdoc *TD) GetAtTypeVocab() string {
	atTypeVocab, found := vocab.ThingClassesMap[tdoc.AtType]
	if !found {
		return tdoc.AtType
	}
	return atTypeVocab.Title
}

// GetEvent returns the Schema for the event or nil if the event doesn't exist
func (tdoc *TD) GetEvent(eventName string) *EventAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	eventAffordance, found := tdoc.Events[eventName]
	if !found {
		return nil
	}
	return eventAffordance
}

// GetForm returns the form for the requested operation
//
// This:
//
// 1. determine the affordance or thing level to use based on the operation
// 2. determine the base path to use
// 3. determine the protocol binding to use from the href (https, mqtt, ...)
// 4. determine which form matches the available protocol binding(s)
//
//	operation is the operation as defined in TD forms
//	name is the name of property, event or action whose form to get
//	protocol to get, eg: http, mqtt, coap
func (tdoc *TD) GetForm(operation string, name string, protocol string) Form {
	var f []Form

	switch operation {
	case
		vocab.WotOpInvokeAction,
		vocab.WotOpCancelAction,
		vocab.WotOpQueryAction,
		vocab.WotOpQueryAllAction,
		"readaction",
		"readallactions":
		aff, _ := tdoc.Actions[name]
		if aff != nil {
			f = aff.Forms
		}

	case
		vocab.WotOpObserveAllProperties,
		vocab.WoTOpObserveProperty,
		vocab.WotOpReadAllProperties,
		vocab.WoTOpReadProperty,
		vocab.WotOpReadMultipleProperties,
		vocab.WotOpUnobserveAllProperties,
		vocab.WoTOpUnobserveProperty,
		vocab.WoTOpWriteProperty,
		vocab.WotOpWriteAllProperties,
		vocab.WotOpWriteMultipleProperties:
		aff, _ := tdoc.Properties[name]
		if aff != nil {
			f = aff.Forms
		}

	case
		vocab.WotOpSubscribeAllEvents,
		vocab.WotOpSubscribeEvent,
		vocab.WotOpUnsubscribeAllEvents,
		vocab.WotOpUnsubscribeEvent,
		"readallevents":
		aff, _ := tdoc.Events[name]
		if aff != nil {
			f = aff.Forms
		}
	}
	if f == nil {
		f = tdoc.Forms
	}
	// find the form for this operation that has the currently used transport type
	for _, form := range f {
		op, found := form["op"]
		if found && op == operation {
			href, found := form.GetHRef()
			if found {
				if strings.HasPrefix(href, protocol) {
					return form
				}
			}
		}
	}
	return nil
}

// GetFormHRef returns the URL of the form and injects the URI variables if provided.
// If the form uses a relative href then prepend it with the base defined
// in the TD.
func (tdoc *TD) GetFormHRef(form Form, uriVars map[string]string) (string, error) {
	href, found := form.GetHRef()
	if !found {
		return "", fmt.Errorf("Form has no href field")
	}
	uri, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	if !uri.IsAbs() {
		href, err = url.JoinPath(tdoc.Base, uri.Path)
	}
	if uriVars != nil {
		href = utils.Substitute(href, uriVars)
	}
	return href, nil
}

// GetProperty returns the Schema and value for the property or nil if its key is not found.
func (tdoc *TD) GetProperty(propName string) *PropertyAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()
	propAffordance, found := tdoc.Properties[propName]
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

// GetAge is a helper function to return the age of the TD.
func (tdoc *TD) GetAge() string {
	modifiedTime, err := dateparse.ParseAny(tdoc.Modified)
	if err != nil {
		return tdoc.Modified
	}
	age := utils.Age(modifiedTime)
	return age
}

// GetUpdated is a helper function to return the formatted time the thing was last updated.
// This uses the time format RFC822 ("02 Jan 06 15:04 MST")
func (tdoc *TD) GetUpdated() string {
	created, err := dateparse.ParseAny(tdoc.Modified)
	if err != nil {
		return tdoc.Modified
	}
	created = created.Local()
	return created.Format(time.RFC822)

}

// GetID returns the ID of the things TD
func (tdoc *TD) GetID() string {
	return tdoc.ID
}

// LoadFromJSON loads this TD from the given JSON encoded string
func (tdoc *TD) LoadFromJSON(tddJSON string) error {
	err := json.Unmarshal([]byte(tddJSON), &tdoc)
	tdoc.EscapeKeys()
	return err
}

// UpdateAction adds a new or replaces an existing action affordance of actionID. Intended for creating TDs.
//
//	name is the name under which to store the action affordance.
//
// This returns the added action affordance
func (tdoc *TD) UpdateAction(name string, affordance *ActionAffordance) *ActionAffordance {
	name = strings.ReplaceAll(name, " ", "-")
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
	name = strings.ReplaceAll(name, " ", "-")
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
	name = strings.ReplaceAll(name, " ", "-")
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Properties[name] = affordance
	return affordance
}

// UpdateTitleDescription sets the title and description of the Thing in the default language
//
//	title is the name of the device, equivalent to the vocab.PropDeviceName property.
//	description is the device product description.
func (tdoc *TD) UpdateTitleDescription(title string, description string) {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Title = title
	tdoc.Description = description
}

// NewTD creates a new Thing Description document with properties, events and actions
//
// Conventions:
// 1. thingID is a URI, starting with the "urn:" prefix as per WoT standard.
// 2. If title is editable then the user should add a property with ID vocab.PropDeviceTitle and update the TD if it is set.
// 3. If description is editable then the user should add a property with ID vocab.PropDeviceDescription and
// update the TD description if it is set.
// 4. the deviceType comes from the vocabulary and has ID vocab.DeviceType<Xyz>
//
// Devices or bindings are not expected to use forms. The form content describes the
// connection protocol which should reference the hub, not the device.
//
//	 Its structure:
//		{
//		     @context: "http://www.w3.org/ns/td",{"ht":"http://hiveot.net/vocab/v..."}
//		     @type: <deviceType>,        // required in HiveOT. See DeviceType vocabulary
//		     id: <thingID>,              // urn:[{prefix}:]{randomID}   required in hiveot
//		     title: string,              // required. Name of the thing
//		     created: <rfc3339>,         // will be the current timestamp. See vocabulary TimeFormat
//		     actions: {name:TDAction, ...},
//		     events:  {name: TDEvent, ...},
//		     properties: {name: TDProperty, ...}
//		}
func NewTD(thingID string, title string, deviceType string) *TD {
	thingID = strings.ReplaceAll(thingID, " ", "-")

	td := TD{
		AtContext: []any{
			WoTTDContext,
			map[string]string{"ht": HiveOTContext},
		},
		AtType:  deviceType,
		Actions: map[string]*ActionAffordance{},

		Created:    time.Now().Format(time.RFC3339),
		Events:     map[string]*EventAffordance{},
		Forms:      nil,
		ID:         thingID,
		Modified:   time.Now().Format(time.RFC3339),
		Properties: map[string]*PropertyAffordance{},

		// security schemas are optional for devices themselves but will be added by the Hub services
		// that provide the TDD.
		Security:            vocab.WoTNoSecurityScheme,
		SecurityDefinitions: make(map[string]SecurityScheme),
		Title:               title,
		updateMutex:         sync.RWMutex{},
	}

	return &td
}
