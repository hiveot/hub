package _go

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenActionClient generates a client function for invoking a Thing action.
//
// This client will marshal the API parameters into an action argument struct and
// invoke the method using the provided messaging transport.
//
// The TD document must be a digital twin received version
func GenActionClient(l *utils.L, td *things.TD) {

	for key, action := range td.Actions {
		GenActionMethod(l, td.ID, key, action)
	}
}

// GenActionMethod generates a client function from an action affordance.
//
//	dtThingID digitwin thingID of the service. This include the agent prefix.
//	key with the service action method.
//	action affordance describing the input and output parameters
func GenActionMethod(l *utils.L, dtThingID string, key string, action *things.ActionAffordance) {
	argsString := "hc hubclient.IHubClient"
	respString := "err error"
	invokeArgs := "nil"
	invokeResp := "nil"

	methodName := Key2ID(key)
	if action.Input != nil {
		argsString = fmt.Sprintf("%s, args %sArgs", argsString, methodName)
		invokeArgs = "&args"
	}
	// add a response struct to arguments
	if action.Output != nil {
		respString = fmt.Sprintf("resp %sResp, %s", methodName, respString)
		invokeResp = "&resp"
	}
	// Function declaration
	l.Indent = 0
	l.Add("")
	l.Add("// %s client method - %s.", methodName, action.Title)
	if len(action.Description) > 0 {
		l.Add("// %s", action.Description)
	}
	l.Add("func %s(%s)(%s){", methodName, argsString, respString)
	l.Indent++
	l.Add("err = hc.Rpc(\"%s\", \"%s\", %s, %s)", dtThingID, key, invokeArgs, invokeResp)
	l.Add("return")
	l.Indent--
	l.Add("}")
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
