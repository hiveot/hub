package goapi

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
)

// GenActionClient generates a client function for invoking an action
// This client will marshal the API parameters into an action argument struct and
// invoke the method using the provided messaging transport.
func GenActionClient(l L, td *things.TD) L {

	for key, action := range td.Actions {
		l = GenActionMethod(l, td.ID, key, action)
	}
	return l
}

// GenActionMethod generates a client function from an action affordance
func GenActionMethod(l L, thingID string, key string, action *things.ActionAffordance) L {
	inputAttrs := GetSchemaAttrs("arg", action.Input)
	outputAttrs := GetSchemaAttrs("result", action.Output)

	methodName := Key2ID(key)
	argString := getArgs(inputAttrs)
	if len(argString) > 0 {
		argString = ", " + argString
	}
	respString := getArgs(outputAttrs)
	if len(respString) > 0 {
		respString = "(" + respString + ", err error)"
	} else {
		respString = "(err error)"
	}
	// Function declaration
	l = l.Add("")
	l = l.Add("// %s %s", methodName, action.Title)
	if len(action.Description) > 0 {
		l = l.Add("// %s", action.Description)
	}
	l = l.Add("func %s(mt api.IMessageTransport%s)%s{", methodName, argString, respString)

	// Instantiate and marshal arguments struct
	l = l.Add("    args := %sArgs {", methodName)
	for _, attr := range inputAttrs {
		// argument names match the args struct names
		l = l.Add("        %s: %s,", attr.AttrName, attr.Key)
	}
	l = l.Add("    }")
	if len(outputAttrs) == 0 {
		// invoke without a response data struct
		// Invoke the message transport
		l = l.Add("    err = mt(\"%s\", \"%s\", &args, nil)", thingID, key)
		l = l.Add("    return err")
	} else {
		// invoke with a response data struct and return the parameters
		l = l.Add("    resp := %sResp{}", methodName)
		// Invoke the message transport
		l = l.Add("    err = mt(\"%s\", \"%s\", &args, &resp)", thingID, key)
		returnParams := ""
		for _, attr := range outputAttrs {
			returnParams += fmt.Sprintf("resp.%s", attr.AttrName)
		}
		// return the result parameter list
		l = l.Add("    return %s, err", returnParams)
	}

	l = l.Add("}")
	return l
}

// getArgs return the function parameter list from the given attributes
func getArgs(attrs []SchemaAttr) string {
	if len(attrs) == 0 {
		return ""
	}
	var attrString = ""
	for _, attr := range attrs {
		if len(attrString) > 0 {
			attrString += ", "
		}
		attrString += fmt.Sprintf("%s %s", attr.Key, attr.AttrType)
	}
	return attrString
}
