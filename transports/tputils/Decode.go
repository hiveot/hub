package tputils

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"strconv"
	"strings"
)

// Decode converts the any-type to the given interface type.
// If the output type is a native type then also consider using one of the DecodeAs...
// methods as these are likely more performant.
// This returns an error if conversion fails.
func Decode(value any, arg interface{}) error {
	if value == nil {
		arg = nil
		return nil
	}
	// the ugly workaround is to marshal/unmarshal using json.
	// TODO: more efficient method to convert the any type to the given type.
	jsonData, _ := jsoniter.Marshal(value)
	return jsoniter.Unmarshal(jsonData, arg)
}

// DecodeAsString converts the value to a string
// if value is already a string then it is returned as-is
// if maxlen is provided then limit the resulting length and add ... if exceeded
func DecodeAsString(value any, maxlen int) string {
	if value == nil {
		return ""
	}
	asString := ""
	switch value.(type) {
	case string:
		asString = value.(string)
	case *string:
		asString = *value.(*string)
	default:
		asString = fmt.Sprintf("%v", value)
	}
	if maxlen <= 0 || len(asString) <= maxlen {
		return asString
	}
	return asString[:maxlen]
}

// DecodeAsBool converts the value to a boolean.
// If value is already a boolean then it is returned as-is.
func DecodeAsBool(value any) bool {
	b := false
	if value == nil {
		return b
	}
	switch value.(type) {
	case bool:
		b = value.(bool)
	case *bool:
		b = *value.(*bool)
	case string:
		b = strings.ToLower(value.(string)) == "true" || value.(string) == "1" || value.(string) == "on"
	case int:
		b = value.(int) != 0
	default:
		slog.Warn("Can't convert value to a boolean", "value", value)
	}
	return b
}

// DecodeAsInt converts the value to an integer.
// This accepts int, int64, *int, bool, uint, float32/64
// If value is already an integer then it is returned as-is.
func DecodeAsInt(value any) int {
	i := 0
	switch value.(type) {
	case bool:
		if value.(bool) {
			i = 1
		}
	case string:
		i64, _ := strconv.ParseInt(value.(string), 10, 64)
		i = int(i64)
	case *int:
		i = *value.(*int)
	case int:
		i = value.(int)
	case uint:
		i = int(value.(uint))
	case float32:
		i = int(value.(float32))
	case float64:
		i = int(value.(float64))
	default:
		slog.Warn("Can't convert value to a integer", "value", value)
	}
	return i
}

// DecodeAsNumber converts the value to a float32 number.
// If value is already a float32 then it is returned as-is.
func DecodeAsNumber(value any) float32 {
	f := float32(0)

	switch value.(type) {
	case float32:
		f = value.(float32)
	case *float32:
		f = *value.(*float32)
	case float64:
		f = float32(value.(float64))
	case *float64:
		f = float32(*value.(*float64))
	case string:
		f32, _ := strconv.ParseFloat(value.(string), 32)
		f = float32(f32)
	}
	return f
}

// DecodeAsObject converts the value to an object.
// If the object is of the same type then it is copied
// otherwise a json marshal/unmarshal is attempted for a deep conversion.
func DecodeAsObject(value any, object interface{}) error {
	if value == nil {
		object = nil
		return nil
	} else {
		serObj, err := jsoniter.Marshal(value)
		if err == nil {
			err = jsoniter.Unmarshal(serObj, object)
		}
		return err
	}
}
