package consumedthing

import (
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
)

// DataSchemaValue represents a value provided by an InteractionOutput.
// This differs from the WoT definition in that it includes type conversions.
type DataSchemaValue struct {
	raw any
}

// Array returns the value as an array
// The result depends on the Schema type
//
//	array: returns array of values as describe ni the Schema
//	boolean: returns a single element true/false
//	bytes: return an array of bytes
//	int: returns a single element with integer
//	object: returns a single element with object
//	string: returns a single element with string
func (v DataSchemaValue) Array() []interface{} {
	objArr := make([]interface{}, 0)
	err := utils.DecodeAsObject(v.raw, &objArr)
	_ = err
	return objArr
}

// Text returns the text representation of the value
func (v DataSchemaValue) Text() string {
	return utils.DecodeAsString(v.raw)
}

func (v DataSchemaValue) Boolean() bool {
	return utils.DecodeAsBool(v.raw)
}
func (v DataSchemaValue) Integer() int {
	return utils.DecodeAsInt(v.raw)
}

// Map returns the value as a name-value map
// Returns nil if no data was provided.
func (v DataSchemaValue) Map() map[string]interface{} {
	o := make(map[string]interface{})
	err := utils.DecodeAsObject(v.raw, &o)
	if err != nil {
		slog.Error("Can't convert value to a map", "value", v.raw)
	}
	return o
}

// Number returns the value as a float32
func (v DataSchemaValue) Number() float32 {
	return utils.DecodeAsNumber(v.raw)
}

// Object decodes the value to the given object.
func (v DataSchemaValue) Object(output interface{}) error {
	return utils.DecodeAsObject(v.raw, output)
}

// Raw returns the raw value
func (v DataSchemaValue) Raw() any {
	return v.raw
}

// NewDataSchemaValue implements a dataschema value
func NewDataSchemaValue(value any) DataSchemaValue {
	return DataSchemaValue{raw: value}
}
