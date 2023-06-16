// Package thing with Schema type definitions for the ExposedThing and ConsumedThing classes
// as described here: https://www.w3.org/TR/wot-thing-description/#sec-data-schema-vocabulary-definition
package thing

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
	AtType string `json:"@type,omitempty"`
	// Provides a human-readable title in the default language
	Title string `json:"title,omitempty"`
	// Provides a multi-language human-readable titles
	Titles []string `json:"titles,omitempty"`
	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`
	// Provides additional nulti-language information
	Descriptions []string `json:"descriptions,omitempty"`

	// Provides a constant value of any type as per data Schema
	Const interface{} `json:"const,omitempty"`

	// Provides a default value of any type as per data Schema
	Default interface{} `json:"default,omitempty"`

	// Allows validation based on a format pattern such as "date-time", "email", "uri", etc.
	// See vocab DataFormXyz "date-time", "email", "uri" (todo)
	Format string `json:"format,omitempty"`

	// Initial value at time of creation
	// this is always a string with optionally a unit
	// not part of the WoT definition but useful for testing and debugging
	InitialValue string `json:"initialValue,omitempty"`

	// OneOf provides constraint of data as one of the given data schemas
	OneOf []interface{} `json:"oneOf,omitempty"`

	// Restricted set of values provided as an array.
	//  for example: ["option1", "option2"]
	Enum []interface{} `json:"enum,omitempty"`

	// Boolean value to indicate whether a property interaction / value is read-only (=true) or not (=false)
	// the value true implies read-only.
	ReadOnly bool `json:"readOnly,omitempty"`

	// Boolean value to indicate whether a property interaction / value is write-only (=true) or not (=false)
	// the value true implies writable but not readable. Intended for secrets such as passwords.
	WriteOnly bool `json:"writeOnly,omitempty"`

	// Type provides JSON based data type,  one of WoTDataTypeNumber, ...object, array, string, integer, boolean or null
	Type string `json:"type,omitempty"`

	// Unit as used in international science, engineering, and business.
	// See vocab UnitNameXyz for units in the vocabulary
	Unit string `json:"unit,omitempty"`

	// ArraySchema with metadata describing data of type Array.
	// https://www.w3.org/TR/wot-thing-description/#arrayschema
	// Used to define the characteristics of an array.
	// Note that in golang a field cannot both be a single or an array of items.
	ArrayItems interface{} `json:"items,omitempty"`
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
	NumberMaximum float64 `json:"maximum,omitempty"`
	// Minimum specifies a minimum numeric value representing a lower limit
	NumberMinimum float64 `json:"minimum,omitempty"`

	// IntegerSchema with metadata describing data of type integer.
	// This Subclass is indicated by the value integer assigned to type in DataSchema instances.
	// Maximum specifies a maximum integer value representing an upper limit
	//IntegerMaximum int `json:"maximum,omitempty"`
	// Minimum specifies a minimum integer value representing a lower limit
	//IntegerMinimum int `json:"minimum,omitempty"`

	// ObjectSchema with metadata describing data of type Object.
	// This Subclass is indicated by the value object assigned to type in DataSchema instances.
	// Properties of Object.
	Properties map[string]DataSchema `json:"properties,omitempty"`
	// Defines which members of the object type are mandatory
	PropertiesRequired []string `json:"required,omitempty"`

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
