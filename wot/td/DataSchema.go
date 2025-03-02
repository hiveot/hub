// Package td with Schema type definitions for the ExposedThing and ConsumedThing classes
// as described here: https://www.w3.org/TR/wot-thing-description/#sec-data-schema-vocabulary-definition
package td

import (
	"github.com/hiveot/hub/wot"
)

// DataSchema with metadata  that describes the data format used. It can be used for validation.
//
// Golang doesn't support dynamic types or subclasses, so DataSchema merges all possible schemas
// including string, number, integer, object, array,...
//
// based on https://www.w3.org/TR/wot-thing-description/#dataschema
type DataSchema struct {
	// JSON-LD keyword to classify the object with semantic tags, if any
	// Intended for grouping and querying similar data, and standardized presentation such as icons
	// For properties this would be the property type, eg temperature, humidity, etc.
	// For action input this is the action parameter type
	// For event data this is the event payload type
	AtType any `json:"@type,omitempty"`
	// Provides a human-readable title in the default language
	Title string `json:"title,omitempty"`
	// Provides a multi-language human-readable titles
	Titles []string `json:"titles,omitempty"`
	// Provides additional (human-readable) information based on a default language
	// this can be a string or array for multi-line description.
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Optional comments for additional multi-line descriptions
	// Not part of the WoT spec as it doesn't support multi-line descriptions
	Comments []string `json:"comments,omitempty"`

	// Provides a constant value of any type as per data Schema
	Const any `json:"const,omitempty"`

	// Provides a default value of any type as per data Schema
	Default any `json:"default,omitempty"`

	// Restricted set of values provided as an array.
	//  for example: ["option1", "option2"]
	Enum []any `json:"enum,omitempty"`

	// Allows validation based on a format pattern such as "date-time", "email", "uri", etc.
	// See vocab DataFormXyz "date-time", "email", "uri" (todo)
	Format string `json:"format,omitempty"`

	// OneOf provides constraint of data as one of the given data schemas
	OneOf []DataSchema `json:"oneOf,omitempty"`

	// Boolean value to indicate whether a property interaction / value is read-only (=true) or not (=false)
	// the value true implies read-only.
	ReadOnly bool `json:"readOnly"`

	// Boolean value to indicate whether a property interaction / value is write-only (=true) or not (=false)
	// the value true implies writable but not readable. Intended for secrets such as passwords.
	WriteOnly bool `json:"writeOnly,omitempty"`

	// Type provides JSON based data type,  one of DataTypeNumber, ...object, array, string, integer, boolean or null
	Type string `json:"type,omitempty"`

	// Reference to an external data schema when type is object
	Schema string `json:"schema,omitempty"`

	// Unit as used in international science, engineering, and business.
	// See vocab UnitNameXyz for units in the vocabulary
	Unit string `json:"unit,omitempty"`

	// ArraySchema with metadata describing data of type Array.
	// https://www.w3.org/TR/wot-thing-description/#arrayschema
	// Used to define the characteristics of an array.
	// Note that in golang a field cannot both be a single or an array of items
	// Right now 'items' has to be a dataschema
	ArrayItems *DataSchema `json:"items,omitempty"`
	// Defines the minimum number of items that have to be in the array
	ArrayMinItems uint `json:"minItems,omitempty"`
	// Defines the maximum number of items that have to be in the array.
	ArrayMaxItems uint `json:"maxItems,omitempty"`

	// BooleanSchema with metadata describing data of type boolean.
	// This Subclass is indicated by the value boolean assigned to type in DataSchema instances.
	// nothing added

	// NumberSchema with metadata describing data of type number.
	// This Subclass is indicated by the value number assigned to type in DataSchema instances.
	// Maximum specifies a maximum numeric value representing an upper limit
	// FIXME: both number and integer schemas have a minimum and maximum field.
	// Different types of course. How to flatten ?
	Maximum float64 `json:"maximum,omitempty"`
	// Minimum specifies a minimum numeric value representing a lower limit
	Minimum float64 `json:"minimum,omitempty"`

	// IntegerSchema with metadata describing data of type integer.
	// This Subclass is indicated by the value integer assigned to type in DataSchema instances.
	// Maximum specifies a maximum integer value representing an upper limit
	//IntegerMaximum int `json:"maximum,omitempty"`
	// Minimum specifies a minimum integer value representing a lower limit
	//IntegerMinimum int `json:"minimum,omitempty"`

	// ObjectSchema with metadata describing data of type Object.
	// This Subclass is indicated by the value object assigned to type in DataSchema instances.
	// Properties of Object.
	Properties map[string]*DataSchema `json:"properties,omitempty"`
	// Defines which members of the object type are mandatory
	Required []string `json:"required,omitempty"`
	// object schema is a map with the data type defined in AdditionalProperties
	// this datatype can be a dataschema or a $ref string
	AdditionalProperties map[string]any `json:"additionalProperties,omitempty"`

	// StringSchema with metadata describing data of type string.
	// This Subclass is indicated by the value string assigned to type in DataSchema instances.
	// MaxLength specifies the maximum length of a string
	StringMaxLength uint `json:"maxLength,omitempty"`
	// MinLength specifies the minimum length of a string
	StringMinLength uint `json:"minLength,omitempty"`
	// Pattern provides a regular expression to express constraints.
	// The regular expression must follow the [ECMA-262] dialect.	optional
	StringPattern string `json:"pattern,omitempty"`
	// ContentEncoding specifies the encoding used to store the contents, as specified in RFC 2054.
	// e.g., 7bit, 8bit, binary, quoted-printable, or base64
	StringContentEncoding string `json:"contentEncoding,omitempty"`
	// ContentMediaType specifies the MIME type of the contents of a string value, as described in RFC 2046.
	// e.g., image/png, or audio/mpeg)
	StringContentMediaType string `json:"contentMediaType,omitempty"`
}

// GetAtTypeString returns the @type field as a string.
// If @type contains an array, the first element is returned.
func (ds *DataSchema) GetAtTypeString() string {
	switch t := ds.AtType.(type) {
	case string:
		return t
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}
	return ""
}

// SetOneOfValues updates the data schema with restricted enum values.
// This uses the 'oneOf' field to allow support title and description of enum values.
// See also the discussion at: https://github.com/w3c/wot-thing-description/issues/997#issuecomment-1865902885
// Values is a set of dataschema values where:
//   - const is the value, required.
//   - title is the human description of the value
//   - description contains an optional elaboration of the value
//   - @type is optional reference to the corresponding vocabulary for this value (if used)
//func (ds *DataSchema) SetOneOfValues(values []DataSchema) *DataSchema {
//	ds.OneOf = values
//	return ds
//}

// IsNative returns true if the type is string, number, boolean
// If type is an object or array this returns false
// Intended for use in input forms
func (ds *DataSchema) IsNative() bool {
	return ds.Type == wot.DataTypeAnyURI ||
		ds.Type == wot.DataTypeBool ||
		ds.Type == wot.DataTypeInteger ||
		ds.Type == wot.DataTypeUnsignedInt ||
		ds.Type == wot.DataTypeNumber ||
		ds.Type == wot.DataTypeString
}
