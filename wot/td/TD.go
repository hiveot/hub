package td

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/url"
	"sort"
	"strings"
)

const WoTTDContext = "https://www.w3.org/2022/wot/td/v1.1"
const HiveOTContext = "https://www.hiveot.net/vocab/v0.1"

// TD contains the Thing Description document
// Its structure is:
//
//	{
//		@context: <WoTTDContext>, {"hiveot":<HiveOTContext>},
//		@type: <deviceType>,
//		id: <thingID>,
//		title: <human description>,  (why is this not a property?)
//		modified: <iso8601>,
//		schemaDefinitions: {...},
//		actions: {actionID: ActionAffordance, ...},
//		events:  {eventID: EventAffordance, ...},
//		properties: {propID: PropertyAffordance, ...}
//		forms: [...]
//	 }
type TD struct {
	// All action-based interaction affordances of the things
	Actions map[string]*ActionAffordance `json:"actions"`

	// Roles that are allowed to use this thing. Default (empty) is all roles.
	Allow []string `json:"allow,omitempty"`

	// JSON-LD keyword to define shorthand names called terms that are used throughout a TD document. Required.
	// in order to add the "hiveot" namespace, the context value can be a string or map
	// Type: anyURO or Array
	AtContext []any `json:"@context"`

	// JSON-LD keyword to label the object with semantic tags (or types).
	// in HiveOT this contains the device type defined in the vocabulary.
	// Intended for grouping and querying similar devices, and standardized presentation such as icons
	AtType any `json:"@type,omitempty"`

	// Base: The base URI that is used for all relative URI references throughout a TD document.
	Base string `json:"base,omitempty"`

	// ISO8601 timestamp this document was first created. See also 'Modified'.
	Created string `json:"created,omitempty"`

	// Roles that are denied to use this thing. Default (empty or 'none') is no roles.
	Deny []string `json:"deny,omitempty"`

	// Describe the device in the default human-readable language.
	// It is recommended to use the product description.
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// All event-based interaction affordances of the things
	Events map[string]*EventAffordance `json:"events"`

	// Form hypermedia controls to describe how an operation can be performed. Forms are serializations of
	// Protocol Bindings. Thing-level forms are used to describe endpoints for a group of interaction affordances.
	Forms []Form `json:"forms,omitempty"`

	// Version information of the TD document (?not the device??)
	//Version VersionInfo `json:"version,omitempty"` // todo

	// ID is the Thing instance identifier.
	// HiveOT prefixes the thing's ID with the publishing agentID to help with uniqueness, separated by colon:
	//  ThingID format used: urn:agentID:deviceID
	ID string `json:"id,omitempty"`

	// links: todo

	// ISO8601 timestamp this document was last modified. See also 'Created'.
	Modified string `json:"modified,omitempty"`

	// Indicates the WoT Profile mechanisms followed by this Thing Description and
	// the corresponding Thing implementation.
	// Type: anyURI or Array of anyURI
	Profile any `json:"profile,omitempty"`

	// All properties-based interaction affordances of the things
	Properties map[string]*PropertyAffordance `json:"properties"`

	SchemaDefinitions map[string]DataSchema `json:"schemaDefinitions,omitempty"`

	// Security is a string or array of security definition names, chosen from those defined
	// in securityDefinitions.
	// In HiveOT security is handled by the Hub.
	// Type: string or array of string
	Security any `json:"security"`

	// Set of named security configurations (definitions only).
	// Not actually applied unless names are used in a security name-value pair. (why is this mandatory then?)
	SecurityDefinitions map[string]SecurityScheme `json:"securityDefinitions"`

	// Title is the name of the thing in human-readable in the default language. Required.
	// The same name is provided through the properties configuration, where it can be changed.
	// This allows to present the TD by its name without having to load its property values.
	Title string `json:"title"`
	// Human-readable titles in the different languages
	Titles map[string]string `json:"titles,omitempty"`

	// Provides information about the TD maintainer as URI scheme (e.g., mailto [RFC6068], tel [RFC3966], https [RFC9112]).
	Support string `json:"support,omitempty"`

	// for use to access writable data, when TD's are updated on the fly by agents.
	//updateMutex sync.RWMutex
}

// AddAction provides a simple way to add an action affordance Schema to the TD.
// This returns the action affordance that can be augmented/modified directly.
// By default this is considered 'unsafe', it affects device state.
//
// If the action accepts input parameters then set the .Data field to a DataSchema instance that
// describes the parameter(s).
//
//	name is the action name under which it is stored in the action affordance map.
//	actionType from the vocabulary or "" if this is a non-standardized action
//	title is the short display title of the action
//	description optional explanation of the action
//	input with dataschema of the action input data, if any
func (tdoc *TD) AddAction(name string, title string, description string,
	input *DataSchema) *ActionAffordance {

	actionAff := &ActionAffordance{
		Title:       title,
		Description: description,
		Input:       input,
		Safe:        false,
	}
	tdoc.UpdateAction(name, actionAff)
	return actionAff
}

// AddDimmerAction is short for adding an action to control a dimmer
// This includes a data schema for integers
//
//	name is the name of the action, unique within the Thing
//func (tdoc *TD) AddDimmerAction(name string) *ActionAffordance {
//	act := tdoc.AddAction(name, "", "",
//		&DataSchema{
//			Type: wot.DataTypeInteger,
//		})
//	act.SetAtType(vocab.ActionDimmer)
//	return act
//}

// AddDimmerEvent is short for adding an event for a dimmer change event
// This includes a data schema for integers
//
//	eventID is the instance ID of the event, unique within the Thing
//func (tdoc *TD) AddDimmerEvent(eventID string) *EventAffordance {
//	aff := tdoc.AddEvent(eventID, "", "",
//		&DataSchema{
//			AtType: vocab.PropSwitchDimmer,
//			Type:   wot.DataTypeInteger,
//		})
//	aff.SetAtType(vocab.PropSwitchDimmer)
//	return aff
//}

// AddEvent provides a simple way to add an event to the TD.
// This returns the event affordance that can be augmented/modified directly.
// To set a known vocabulary @type, use setVocabType on the result
//
// If the event returns data then set the .Data field to a DataSchema instance that describes it.
//
//	eventName is the name under which it is stored in the affordance map.
//	title is the short display title of the event
//	schema optional event data schema or nil if the event doesn't carry any data
func (tdoc *TD) AddEvent(
	eventName string, title string, description string, schema *DataSchema) *EventAffordance {

	if eventName == "" {
		slog.Error("AddEvent: Missing event name for thing", "thingID", tdoc.ID)
		return nil
	}
	evAff := &EventAffordance{
		Title:       title,
		Description: description,
		//Data:        *schema,
	}
	evAff.Data = schema

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
//	dataType is the type of data the property holds, DataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *TD) AddProperty(propName string, title string, description string, dataType string) *PropertyAffordance {
	prop := &PropertyAffordance{
		DataSchema: DataSchema{
			Title:       title,
			Type:        dataType,
			Description: description,
			ReadOnly:    true,
			//InitialValue: initialValue,
		},
	}
	tdoc.UpdateProperty(propName, prop)
	return prop
}

// AddPropertyAsString is short for adding a read-only string property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsString(
	propName string, title string, description string) *PropertyAffordance {

	return tdoc.AddProperty(propName, title, description, wot.DataTypeString)
}

// AddPropertyAsBool is short for adding a read-only boolean property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsBool(
	propName string, title string, description string) *PropertyAffordance {

	return tdoc.AddProperty(propName, title, description, wot.DataTypeBool)
}

// AddPropertyAsInt is short for adding a read-only integer property
//
//	propType describes the type of property in HiveOT vocabulary if available, or "" if this is a non-standard property.
func (tdoc *TD) AddPropertyAsInt(
	propName string, title string, description string) *PropertyAffordance {

	return tdoc.AddProperty(propName, title, description, wot.DataTypeInteger)
}

// AsMap returns the TD document as a key-value map
func (tdoc *TD) AsMap() map[string]interface{} {
	//tdoc.updateMutex.RLock()
	//defer tdoc.updateMutex.RUnlock()

	var asMap map[string]interface{}
	asJSON, _ := jsoniter.MarshalToString(tdoc)
	_ = jsoniter.UnmarshalFromString(asJSON, &asMap)
	return asMap
}

// EscapeKeys replace spaces in property, event and action keys with dashes
func (tdoc *TD) EscapeKeys() {
	// NOTE: this does not modify links and refs. Those must be valid.
	newProps := make(map[string]*PropertyAffordance)
	for k, v := range tdoc.Properties {
		kesc := strings.ReplaceAll(k, " ", "_")
		newProps[kesc] = v
	}
	tdoc.Properties = newProps

	newEvents := make(map[string]*EventAffordance)
	for k, v := range tdoc.Events {
		kesc := strings.ReplaceAll(k, " ", "_")
		newEvents[kesc] = v
	}
	tdoc.Events = newEvents

	newActions := make(map[string]*ActionAffordance)
	for k, v := range tdoc.Actions {
		kesc := strings.ReplaceAll(k, " ", "_")
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
	//tdoc.updateMutex.RLock()
	//defer tdoc.updateMutex.RUnlock()

	actionAffordance, found := tdoc.Actions[actionName]
	if !found {
		return nil
	}
	return actionAffordance
}

// GetEvent returns the Schema for the event or nil if the event doesn't exist
func (tdoc *TD) GetEvent(eventName string) *EventAffordance {
	//tdoc.updateMutex.RLock()
	//defer tdoc.updateMutex.RUnlock()

	eventAffordance, found := tdoc.Events[eventName]
	if !found {
		return nil
	}
	return eventAffordance
}

// GetForms returns the forms for the requested operation.
// The caller still has to find the matching protocol based on url and subprotocol field.
//
// This first checks for forms in the requested operation by affordance name,
// and appends the global forms.
//
// FIXME: make this wot compatible. The top level forms seem to play a different
// role than the one in the affordances. The other problem is that the sheer
// amount of forms would bloat the TD so I have trouble accepting this approach.
//
//	operation is the operation as defined in TD forms
//	name is the name of property, event or action whose form to get or "" for the TD level operations
func (tdoc *TD) GetForms(operation string, name string) []Form {
	var availableForms []Form
	var opForms []Form = make([]Form, 0)

	if name != "" {
		// get the form from the affordance if a name is given

		switch operation {
		case
			wot.OpInvokeAction,
			wot.OpCancelAction,
			wot.OpQueryAction,
			wot.OpQueryAllActions:
			aff, _ := tdoc.Actions[name]
			if aff != nil {
				availableForms = aff.Forms
			}

		case
			wot.OpObserveAllProperties,
			wot.OpObserveProperty,
			wot.OpReadAllProperties,
			wot.OpReadProperty,
			wot.OpUnobserveAllProperties,
			wot.OpUnobserveProperty,
			wot.OpWriteProperty,
			wot.OpWriteMultipleProperties:
			aff, _ := tdoc.Properties[name]
			if aff != nil {
				availableForms = aff.Forms
			}

		case
			wot.OpSubscribeAllEvents,
			wot.OpSubscribeEvent,
			wot.OpUnsubscribeAllEvents,
			wot.OpUnsubscribeEvent:
			aff, _ := tdoc.Events[name]
			if aff != nil {
				availableForms = aff.Forms
			}
		}

		if availableForms != nil {
			// find the form for this operation that has the currently used transport type
			for _, form := range availableForms {
				op, found := form["op"]
				if found && op == operation {
					opForms = append(opForms, form)
				}
			}
		}
	}
	// still not found? okay check the top level form
	for _, form := range tdoc.Forms {
		op, found := form["op"]
		if found && op == operation {
			opForms = append(opForms, form)
		}
	}
	return opForms
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
		href = tdoc.Substitute(href, uriVars)
	}
	return href, nil
}

// GetProperty returns the Schema and value for the property or nil if its key is not found.
func (tdoc *TD) GetProperty(propName string) *PropertyAffordance {
	//tdoc.updateMutex.RLock()
	//defer tdoc.updateMutex.RUnlock()
	propAffordance, found := tdoc.Properties[propName]
	if !found {
		return nil
	}
	return propAffordance
}

// GetPropertyOfVocabType returns the first property affordance with the given @type
// This returns the property ID and the property affordances, or nil if not found
func (tdoc *TD) GetPropertyOfVocabType(vocabType string) (string, *PropertyAffordance) {
	//tdoc.updateMutex.RLock()
	//defer tdoc.updateMutex.RUnlock()
	for propID, prop := range tdoc.Properties {
		if prop.AtType == vocabType {
			return propID, prop
		}
	}
	return "", nil
}

// GetID returns the ID of the things TD
func (tdoc *TD) GetID() string {
	return tdoc.ID
}

// LoadFromJSON loads this TD from the given JSON encoded string
//func (tdoc *TD) LoadFromJSON(tddJSON string) error {
//	err := jsoniter.UnmarshalFromString(tddJSON, &tdoc)
//	tdoc.EscapeKeys()
//	return err
//}

// SetForms replaces the top level forms section of the TD
func (tdoc *TD) SetForms(formList []Form) {
	//tdoc.updateMutex.Lock()
	//defer tdoc.updateMutex.Unlock()
	tdoc.Forms = formList
}

// Substitute substitutes the variables in a string
// Variables are define with curly brackets, eg: "this is a {variableName}"
func (tdoc *TD) Substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		stringVar := "{" + k + "}"
		s = strings.Replace(s, stringVar, v, -1)
	}
	return s
}

// UpdateAction adds a new or replaces an existing action affordance of actionID. Intended for creating TDs.
//
//	name is the name under which to store the action affordance.
//
// This returns the added action affordance
func (tdoc *TD) UpdateAction(name string, affordance *ActionAffordance) *ActionAffordance {
	name = strings.ReplaceAll(name, " ", "_")
	//tdoc.updateMutex.Lock()
	//defer tdoc.updateMutex.Unlock()
	if tdoc.Actions == nil {
		tdoc.Actions = make(map[string]*ActionAffordance)
	}
	tdoc.Actions[name] = affordance
	return affordance
}

// UpdateEvent adds a new or replaces an existing event affordance of eventID. Intended for creating TDs
//
//	name is the ID under which to store the event affordance.
//
// This returns the added event affordance.
func (tdoc *TD) UpdateEvent(name string, affordance *EventAffordance) *EventAffordance {
	name = strings.ReplaceAll(name, " ", "_")
	//tdoc.updateMutex.Lock()
	//defer tdoc.updateMutex.Unlock()
	if tdoc.Events == nil {
		tdoc.Events = make(map[string]*EventAffordance)
	}
	tdoc.Events[name] = affordance
	return affordance
}

// UpdateProperty adds or replaces a property affordance in the TD. Intended for creating TDs
//
//	name is the name under which to store the property affordance.
//
// This returns the added affordance to support chaining
func (tdoc *TD) UpdateProperty(name string, affordance *PropertyAffordance) *PropertyAffordance {
	name = strings.ReplaceAll(name, " ", "_")
	//tdoc.updateMutex.Lock()
	//defer tdoc.updateMutex.Unlock()
	if tdoc.Properties == nil {
		tdoc.Properties = make(map[string]*PropertyAffordance)
	}
	tdoc.Properties[name] = affordance
	return affordance
}

// UpdateTitleDescription sets the title and description of the Thing in the default language
//
//	title is the name of the device, equivalent to the vocab.PropDeviceName property.
//	description is the device product description.
func (tdoc *TD) UpdateTitleDescription(title string, description string) {
	//tdoc.updateMutex.Lock()
	//defer tdoc.updateMutex.Unlock()
	tdoc.Title = title
	tdoc.Description = description
}

// NewTD creates a new Thing Description document with properties, events and actions
//
// Conventions:
// 1. thingID is a URI, starting with the "urn:" prefix as per WoT standard.
// 2. Agents should add a property wot.WoTTitle and update the TD if it is set.
// 3. Agents should add a property wot.WoTDescription and update the TD description if it is set.
// 4. the deviceType comes from the vocabulary and has ID vocab.DeviceType<Xyz>
//
// Devices or bindings are not expected to use forms. The form content describes the
// connection protocol which should reference the hub, not the device.
//
//	 Its structure:
//		{
//		     @context: "http://www.w3.org/ns/td",{"hiveot":"http://hiveot.net/vocab/v..."}
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
			map[string]string{"hiveot": HiveOTContext},
		},
		AtType:  deviceType,
		Actions: map[string]*ActionAffordance{},

		Created:    utils.FormatNowUTCMilli(),
		Events:     map[string]*EventAffordance{},
		Forms:      nil,
		ID:         thingID,
		Modified:   utils.FormatNowUTCMilli(),
		Properties: map[string]*PropertyAffordance{},

		// security schemas are optional for devices themselves but will be added by the Hub services
		// that provide the TDD.
		Security:            wot.WoTNoSecurityScheme,
		SecurityDefinitions: make(map[string]SecurityScheme),
		Title:               title,
		//updateMutex:         sync.RWMutex{},
	}

	return &td
}

// SortThingsByID as the name suggests sorts the things in the given slice
func SortThingsByID(tds []*TD) {
	sort.Slice(tds, func(i, j int) bool {
		tdI := tds[i]
		tdJ := tds[j]
		return tdI.ID < tdJ.ID
	})
}

// SortThingsByTitle as the name suggests sorts the things in the given slice
func SortThingsByTitle(tds []*TD) {
	sort.Slice(tds, func(i, j int) bool {
		tdI := tds[i]
		tdJ := tds[j]
		return tdI.Title < tdJ.Title
	})
}
