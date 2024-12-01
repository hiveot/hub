package utils

import "strings"

// Substitute substitutes the variables in a string
// Variables are define with curly brackets, eg: "this is a {variableName}"
func Substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		stringVar := "{" + k + "}"
		s = strings.Replace(s, stringVar, v, -1)
	}
	return s
}
