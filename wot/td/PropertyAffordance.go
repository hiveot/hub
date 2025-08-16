// Package things with API interface definitions for the ExposedThing and ConsumedThing classes
package td

// PropertyAffordance metadata that defines Thing properties
// This is a Subclass of the InteractionAffordance Class and the DataSchema Class.
// Note: https://github.com/w3c/wot-thing-description/issues/1390
// The spec simply cannot be implemented in golang without dynamic types.
// PropertyAffordance must be able to have a Schema based on the type, not just DataSchema, as
// a property can be any of the types integer, boolean, object, array, number...
type PropertyAffordance struct {
	DataSchema

	// This property contains a sensor value. This is not a WoT attribute.
	IsSensor bool `json:"isSensor,omitempty"`

	// This property contains an actuator state. This is not a WoT attribute.
	IsActuator bool `json:"isActuator,omitempty"`

	// Since goland doesn't support inheritance the interaction affordance fields are defined below
	// @type, title(s) and description(s) are already defined in the embedded DataSchema struct

	// Form hypermedia controls to describe how an operation can be performed
	// Forms are serializations of Protocol Bindings. Don't omit if empty as forms are mandatory.
	Forms []Form `json:"forms"`

	// Define URI template variables according to [RFC6570] as collection based on DataSchema declarations.
	// ... right
	UriVariables map[string]DataSchema `json:"uriVariables,omitempty"`

	// A hint that indicates whether Servients hosting the Thing and Intermediaries should provide
	// a Protocol Binding that supports the observeproperty and unobserveproperty operations for
	// this Property.
	// This is implied for Things that are using the message bus and not documented here.
	//Observable bool `json:"observable,omitempty" default:"false"`

	// Optional nested properties. Map with PropertyAffordance
	Properties map[string]PropertyAffordance `json:"properties,omitempty"`
}

// AddForm adds an interaction form to the property affordance
// this is not thread-safe.
func (aff *PropertyAffordance) AddForm(form Form) {
	aff.Forms = append(aff.Forms, form)
}

// SetAtType sets the property @type field from the HT vocabulary
func (aff *PropertyAffordance) SetAtType(atType string) *PropertyAffordance {
	aff.AtType = atType
	return aff
}
