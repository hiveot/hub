package goapi

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"strings"
)

// This file contains naming helper functions
// * Key2ID turns a TD key into a valid identifier
// * ToTitle turns words in a string into a valid title case
// * GetSchemaType converts a dataschema type into a golang type

// GetSchemaType returns the golang type of a dataschema type
func GetSchemaType(ds *things.DataSchema) string {
	switch ds.Type {
	case vocab.WoTDataTypeAnyURI:
		return "string"
	case vocab.WoTDataTypeArray:
		// the actual type is in a dataschema under 'items'
		if ds.ArrayItems != nil {
			arrayType := GetSchemaType(ds.ArrayItems)
			return "[]" + arrayType
		} else {
			// unknown type
			return "[]interface{}"
		}
	case vocab.WoTDataTypeDateTime:
		return "string"
	case vocab.WoTDataTypeBool:
		return "bool"
	case vocab.WoTDataTypeInteger:
		return "int"
	case vocab.WoTDataTypeNumber:
		return "float64"
	case vocab.WoTDataTypeString:
		return "string"
	case vocab.WoTDataTypeUnsignedInt:
		return "uint64"
	case vocab.WoTDataTypeObject:
		return "object"
	default:
		return "string"
	}
}

// Key2ID turns a TD key into a valid identifier string
// This removes invalid characters and capitalizes the result
func Key2ID(key string) string {
	keyAsBytes := []byte(key)
	// keep only valid identifier chars
	n := 0
	for _, b := range keyAsBytes {
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') ||
			b == '_' {
			keyAsBytes[n] = b
			n++
		}
	}
	// first char must be upper case
	id := ToTitle(string(keyAsBytes[:n]))
	return id
}

// ToTitle convers the given string to title case
func ToTitle(s string) string {
	words := strings.Fields(s)
	for n, word := range words {
		words[n] = strings.Title(word)
	}
	// join without spaces
	s2 := strings.Join(words, "")
	return s2
}
