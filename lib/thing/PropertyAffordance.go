// Package thing with API interface definitions for the ExposedThing and ConsumedThing classes
package thing

// PropertyAffordance metadata that defines Thing properties
// This is a Subclass of the InteractionAffordance Class and the DataSchema Class.
// Note: https://github.com/w3c/wot-thing-description/issues/1390
// The spec simply cannot be implemented in golang without dynamic types.
// PropertyAffordance must be able to have a Schema based on the type, not just DataSchema, as
// a property can be any of the types integer, boolean, object, array, number...
type PropertyAffordance struct {
	DataSchema

	// Since goland doesn't support inheritance the interaction affordance fields are defined below
	// @type, title(s) and description(s) are already defined in the embedded DataSchema struct

	// Form hypermedia controls to describe how an operation can be performed
	// Forms are serializations of Protocol Bindings.
	Forms []Form `json:"forms,omitempty"`

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
