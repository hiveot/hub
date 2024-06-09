package utils

import (
	"fmt"
	"reflect"
)

// ToString converts the data to a string.
// If data is a string it is returned as-as
// If data is a number it is converted to a string using %v
// If data is an array it is converted to a multi-line with \n between elements
//
// Intended to allow multi-line text
func ToString(data any) string {

	dType := reflect.TypeOf(data)
	switch dType.Kind() {
	case reflect.Slice:
		reply := ""
		l := data.([]any)
		for _, item := range l {
			reply += ToString(item) + "\n"
		}
		return reply
	case reflect.String:
		return data.(string)
	default:
		return fmt.Sprintf("%v", data)
	}
}
