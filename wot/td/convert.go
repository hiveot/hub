package td

import (
	"errors"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"strconv"
	"strings"
)

// Helper methods for converting TD, event, property and action values to and from text.
// Intended for assisting conversion between text and native formats.

// UnmarshalTD unmarshals a JSON encoded TD
func UnmarshalTD(tdJSON string) (td *TD, err error) {
	td = &TD{}
	err = jsoniter.UnmarshalFromString(tdJSON, td)
	return td, err
}

func UnmarshalTDList(tdListJSON []string) (tdList []*TD, err error) {
	tdList = make([]*TD, 0, len(tdListJSON))
	for _, tdJson := range tdListJSON {
		td := TD{}
		err = jsoniter.UnmarshalFromString(tdJson, &td)
		if err == nil {
			tdList = append(tdList, &td)
		}
	}
	return tdList, err
}

// ConvertToNative converts the string value to native type based on the given data schema
// this converts int, float, and boolean
// if the dataschema is an object or an array then strVal is assumed to be json encoded
func ConvertToNative(strVal string, dataSchema *DataSchema) (val any, err error) {
	if strVal == "" {
		// nil value boolean input are always treated as false.
		if dataSchema.Type == wot.DataTypeBool {
			return false, nil
		}
		return nil, nil
	} else if dataSchema == nil {
		slog.Error("ConvertToNative: nil DataSchema")
		return nil, errors.New("Nil DataSchema")
	}
	switch dataSchema.Type {
	case wot.DataTypeBool:
		// ParseBool is too restrictive
		lowerVal := strings.ToLower(strVal)
		val = false
		if strVal == "1" || lowerVal == "true" || lowerVal == "on" {
			val = true
		}
		break
	case wot.DataTypeArray:
		err = jsoniter.UnmarshalFromString(strVal, &val)
		break
	case wot.DataTypeDateTime:
		val, err = dateparse.ParseAny(strVal)
		break
	case wot.DataTypeInteger:
		val, err = strconv.ParseInt(strVal, 10, 64)
		break
	case wot.DataTypeNumber:
		val, err = strconv.ParseFloat(strVal, 64)
		break
	case wot.DataTypeUnsignedInt:
		val, err = strconv.ParseUint(strVal, 10, 64)
		break
	case wot.DataTypeObject:
		err = jsoniter.UnmarshalFromString(strVal, &val)
		break
	default:
		val = strVal
		break
	}
	return val, err
}
