package tdd2go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenActionStructs generates argument and response structs for actions defined in the TD.
// This returns and array of lines of code or an error
func GenActionStructs(l *utils.L, serviceTitle string, td *things.TD) {
	l.Indent = 0
	l.Add("//--- Argument and Response struct for action of Thing '%s' ---", td.ID)
	l.Add("")

	actionKeys := utils.OrderedMapKeys(td.Actions)
	for _, key := range actionKeys {
		action := td.Actions[key]
		methodName := Key2ID(key)
		// define a constants for this action method name
		l.Add("const %s%sMethod = \"%s\"", serviceTitle, methodName, key)
		l.Add("")
		// define structs for action method arguments and responses
		GenActionArgs(l, serviceTitle, key, action)
		GenActionResp(l, serviceTitle, key, action)
	}
}

// GenActionArgs generates the arguments struct of the given action, if any
// Argument structs are named the '{key}'Args where key is modified to remove invalid chars
func GenActionArgs(l *utils.L, serviceTitle string, key string, action *things.ActionAffordance) {

	// no need if the input is not a struct
	if action.Input == nil || action.Input.Type != "object" {
		return
	}
	typeName := Key2ID(key)
	l.Indent = 0
	l.Add("// %s%sArgs defines the arguments of the %s function", serviceTitle, typeName, key)
	l.Add("// %s - %s", action.Title, action.Description)
	GenDescription(l, action.Input.Description, action.Input.Comments)
	l.Add("type %s%sArgs struct {", serviceTitle, typeName)
	// input is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Input'
	//attrList := GetSchemaAttrs("Input", action.Input)
	l.Indent++
	GenDataSchemaFields(l, "input", action.Input)
	l.Indent--
	l.Add("}")
	l.Add("")
}

// GenActionResp generates the response struct of the given action, if any.
// Response structs are named the {key}Resp where key is modified to remove invalid chars
func GenActionResp(l *utils.L, serviceTitle string, key string, action *things.ActionAffordance) {
	// no need if the output is not a struct
	if action.Output == nil || action.Output.Type != "object" {
		return
	}
	typeName := Key2ID(key)
	l.Indent = 0
	l.Add("// %s%sResp defines the response of the %s function", serviceTitle, typeName, key)
	l.Add("// %s - %s", action.Title, action.Description)
	GenDescription(l, action.Output.Description, action.Output.Comments)
	l.Add("type %s%sResp struct {", serviceTitle, typeName)
	// output is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Output'
	//attrList := GetSchemaAttrs("output", action.Output)
	l.Indent++
	GenDataSchemaFields(l, "output", action.Output)
	//GenDataSchemaParams(l, attrList)
	l.Indent--
	l.Add("}")
	l.Add("")
}
