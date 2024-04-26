package goapi

import "github.com/hiveot/hub/lib/things"

// GenActionStructs generates argument and response structs for actions defined in the TD
// This returns and array of lines of code or an error
func GenActionStructs(l L, td *things.TD) L {
	l = l.Add("// Argument and Response struct for action of Thing '%s'", td.ID)
	l = l.Add("")

	for key, action := range td.Actions {
		l = GenActionArgs(l, key, action)
		l = GenActionResp(l, key, action)
	}
	return l
}

// GenActionArgs generates the arguments struct of the given action, if any
// Argument structs are named the '{key}'Args where key is modified to remove invalid chars
func GenActionArgs(l L, key string, action *things.ActionAffordance) L {

	if action.Input == nil {
		return l
	}
	typeName := Key2ID(key)
	l = l.Add("// %sArgs defines the arguments of the %s function", typeName, typeName)
	l = l.Add("// %s - %s", action.Title, action.Description)
	l = l.Add("type %sArgs struct {", typeName)
	// input is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Input'
	attrList := GetSchemaAttrs("Input", action.Input)
	l = GenDataSchemaParams(l, attrList)
	l = l.Add("}")
	l = l.Add("")
	return l
}

// GenActionResp generates the response struct of the given action, if any.
// Response structs are named the {key}Resp where key is modified to remove invalid chars
func GenActionResp(l L, key string, action *things.ActionAffordance) L {

	if action.Output == nil {
		return l
	}
	typeName := Key2ID(key)
	l = l.Add("// %sResp defines the response of the %s function", typeName, typeName)
	l = l.Add("// %s - %s", action.Title, action.Description)
	l = l.Add("type %sResp struct {", typeName)
	// output is a dataschema which can be a native value or an object with multiple fields
	// if this is a native value then name it 'Output'
	attrList := GetSchemaAttrs("Result", action.Output)
	l = GenDataSchemaParams(l, attrList)
	l = l.Add("}")
	l = l.Add("")
	return l
}
