// Package things with handling of property, event and action values
package things

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// InteractionOutput to expose the data returned from WoT Interactions to applications.
// Use NewInteractionOutput to initialize
type InteractionOutput struct {
	// Schema describing the data from property, event or action affordance
	Schema *DataSchema
	// raw data from the interaction as described by the Schema
	serEncoded []byte
	// decoded data in its native format, eg string, int, array, object
	Value interface{} `json:"value"`
}

//// Value returns the parsed value of the interaction
//func (io *InteractionOutput) Value() interface{} {
//	return io.value
//}

// ValueAsArray returns the value as an array
// The result depends on the Schema type
//
//	array: returns array of values as describe ni the Schema
//	boolean: returns a single element true/false
//	bytes: return an array of bytes
//	int: returns a single element with integer
//	object: returns a single element with object
//	string: returns a single element with string
func (io *InteractionOutput) ValueAsArray() []interface{} {
	obj := make([]interface{}, 0)
	_ = ser.Unmarshal(io.serEncoded, &obj)
	return obj
}

// ValueAsString returns the value as a string
func (io *InteractionOutput) ValueAsString() string {
	var s interface{} = io.Value //"" //string(io.serEncoded)
	if io.serEncoded != nil {
		err := ser.Unmarshal(io.serEncoded, &s)
		if err != nil {
			slog.Error("Can't convert value to a string", "value", io.serEncoded)
		}
	}
	asString := fmt.Sprint(s)
	return asString
}

// ValueAsBoolean returns the value as a boolean
func (io *InteractionOutput) ValueAsBoolean() bool {
	b := false
	err := ser.Unmarshal(io.serEncoded, &b)
	if err != nil {
		slog.Error("Can't convert value to a boolean", "value", io.serEncoded)
	}
	return b
}

// ValueAsInt returns the value as an integer
func (io *InteractionOutput) ValueAsInt() int {
	i := 0
	// special case converting booleans
	if io.Value == "true" || io.Value == true {
		i = 1
	} else if io.Value == "false" || io.Value == false {
		i = 0
	} else {
		err := ser.Unmarshal(io.serEncoded, &i)
		if err != nil {
			slog.Error("Can't convert value to a int", "value", io.serEncoded)
		}
	}
	return i
}

// ValueAsMap returns the value as a key-value map
// Returns nil if no data was provided.
func (io *InteractionOutput) ValueAsMap() map[string]interface{} {
	o := make(map[string]interface{})
	err := ser.Unmarshal(io.serEncoded, &o)
	if err != nil {
		slog.Error("Can't convert value to a map", "value", io.serEncoded)
	}
	return o
}

// NewInteractionOutputFromJson creates a new interaction output for reading output data
// @param serEncoded is raw data that will be decoded using the given Schema
// @param Schema describes the value. nil in case of unknown Schema
func NewInteractionOutputFromJson(jsonEncoded []byte, schema *DataSchema) *InteractionOutput {
	var err error
	var val interface{}
	if schema != nil && schema.Type == vocab.WoTDataTypeObject {
		// If this is an object use a map
		val := make(map[string]interface{})
		err = json.Unmarshal(jsonEncoded, &val)
	} else {
		var sVal interface{}
		err = json.Unmarshal(jsonEncoded, &sVal)
		if err == nil {
			val = sVal
		} else {
			// otherwise keep native type in its string format
			val = string(jsonEncoded)
		}
	}
	if err != nil {
		slog.Error("Error unmarshalling", "err", err)
	}
	io := &InteractionOutput{
		serEncoded: jsonEncoded,
		Schema:     schema,
		Value:      val,
	}
	return io
}

// NewInteractionOutput creates a new interaction output from object data
// data is native that will be ser encoded using the given Schema
// Schema describes the value. nil in case of unknown Schema
func NewInteractionOutput(data interface{}, schema *DataSchema) *InteractionOutput {
	serEncoded, err := ser.Marshal(data)
	if err != nil {
		slog.Error("Unable to marshal data", data)
	}
	io := &InteractionOutput{
		serEncoded: serEncoded,
		Schema:     schema,
		Value:      data,
	}
	return io
}
