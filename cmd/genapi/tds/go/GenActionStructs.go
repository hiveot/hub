package _go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenActionStructs generates argument and response structs for actions defined in the TD
// This returns and array of lines of code or an error
func GenActionStructs(l *utils.L, td *things.TD) {
	l.Add("// Argument and Response struct for action of Thing '%s'", td.ID)
	l.Add("")

	for key, action := range td.Actions {
		GenActionArgs(l, key, action)
		GenActionResp(l, key, action)
	}
}

// GenActionArgs generates the arguments struct of the given action, if any
// Argument structs are named the '{key}'Args where key is modified to remove invalid chars
func GenActionArgs(l *utils.L, key string, action *things.ActionAffordance) {

	if action.Input == nil {
		return
	}
	typeName := Key2ID(key)
	l.Add("// %sArgs defines the arguments of the %s function", typeName, typeName)
	l.Add("// %s - %s", action.Title, action.Description)
	l.Add("type %sArgs struct {", typeName)
	// input is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Input'
	attrList := GetSchemaAttrs("Input", action.Input)
	GenDataSchemaParams(l, attrList)
	l.Add("}")
	l.Add("")
}

// GenActionResp generates the response struct of the given action, if any.
// Response structs are named the {key}Resp where key is modified to remove invalid chars
func GenActionResp(l *utils.L, key string, action *things.ActionAffordance) {

	if action.Output == nil {
		return
	}
	typeName := Key2ID(key)
	l.Add("// %sResp defines the response of the %s function", typeName, typeName)
	l.Add("// %s - %s", action.Title, action.Description)
	l.Add("type %sResp struct {", typeName)
	// output is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Output'
	attrList := GetSchemaAttrs("Result", action.Output)
	GenDataSchemaParams(l, attrList)
	l.Add("}")
	l.Add("")
}