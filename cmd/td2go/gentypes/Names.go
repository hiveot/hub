package gentypes

import (
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"strings"
	"unicode"
	"unicode/utf8"
)

// This file contains naming helper functions
// * Name2ID turns a TD key into a valid identifier
// * ToTitle turns words in a string into a valid title case
// * GoTypeFromSchema converts a dataschema type into a golang type

// GoTypeFromSchema returns the golang type of a dataschema type,
// or the non-standard type if this not a WoT type.
//
// If the type is an object and the 'schema' field contains a type name then use
// the schema name as the type.
//
// return the schema value as the type.
func GoTypeFromSchema(ds *td.DataSchema) string {
	switch ds.Type {
	case wot.DataTypeAnyURI:
		return "string"
	case wot.DataTypeArray:
		// the actual type is in a dataschema under 'items'
		if ds.ArrayItems != nil {
			arrayType := GoTypeFromSchema(ds.ArrayItems)
			return "[]" + arrayType
		} else {
			// unknown type
			return "[]interface{}"
		}
	case wot.DataTypeDateTime:
		return "string"
	case wot.DataTypeBool:
		return "bool"
	case wot.DataTypeInteger:
		return "int"
	case wot.DataTypeNumber:
		return "float64"
	case wot.DataTypeString:
		return "string"
	case wot.DataTypeUnsignedInt:
		return "uint64"
	case wot.DataTypeObject:
		if ds.Schema != "" {
			// Only local references are supported
			return ToTitle(ds.Schema)
		} else if ds.Properties != nil {
			// support map types
			if ds.Properties[""] != nil {
				// this is a map
				mapType := ds.Properties[""]
				if mapType.Schema != "" {
					return "map[string]" + mapType.Schema
				} else {
					return "map[string]" + mapType.Type
				}
			}
			return "nested properties not supported"
		} else {
			return "map[string]interface{}"
		}
	case "":
		return "any"
	default:
		// inject the given type. This is not json-schema standard
		return ds.Type
	}
}

// Name2ID turns a TD name into a valid identifier string
// This removes invalid characters and capitalizes the result
func Name2ID(key string) string {
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

// ToTitle converts each word in the given string to title case
func ToTitle(s string) string {
	words := strings.Fields(s)
	for n, word := range words {
		words[n] = strings.Title(word)
	}
	// join without spaces
	s2 := strings.Join(words, "")
	return s2
}

// FirstToLower converts the first character of a string to lower case
// credits: https://stackoverflow.com/questions/75988064/make-first-letter-of-string-lower-case-in-golang
func FirstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}
