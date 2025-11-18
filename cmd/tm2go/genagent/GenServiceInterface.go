package genagent

import (
	"fmt"
	"regexp"

	"github.com/hiveot/hivehub/cmd/tm2go/gentypes"
	"github.com/hiveot/hivekitgo/utils"
	"github.com/hiveot/hivekitgo/wot/td"
)

// GenServiceInterface generates the interface the service has to implement.
//
//	agentID is this package name, eg: the agent for this service
//	serviceID is the ThingID of the service capitalized
func GenServiceInterface(l *gentypes.SL, agentID, serviceID string, tdi *td.TD) {
	// ServiceType is the interface of the service. Interface names start with 'I'
	interfaceName := "I" + serviceID + "Service"
	l.Add("")
	l.Add("// %s defines the interface of the '%s' service", interfaceName, serviceID)
	l.Add("//")
	l.Add("// This defines a method for each of the actions in the TD. ")
	l.Add("// ")
	l.Add("type %s interface {", interfaceName)

	sortedActionNames := utils.OrderedMapKeys(tdi.Actions)
	for _, name := range sortedActionNames {
		action := tdi.Actions[name]
		GenInterfaceMethod(l, serviceID, name, action)
	}
	l.Add("}")
}

// GenInterfaceMethod adds a method definition for an action in the TD.
//
// > methodName( args ArgsType ) (resp RespType, err error)
//
// TODO: this code to generate a method signature is similar to that in
// GenServiceConsumer.GenActionMethod
//
// The generated method arguments are the senderID and the value.
// The value is either a native type or a struct, based on the TDD definition
func GenInterfaceMethod(l *gentypes.SL, serviceTitle string, name string, action *td.ActionAffordance) {

	// 1. build the input arguments. All methods receive the sender client ID.
	methodName := gentypes.Name2ID(name)
	argsString := "senderID string"

	// add the service input parameters
	if action.Input != nil {
		argName := getParamName("args", action.Input)
		goType := ""
		if action.Input.Type == "object" && action.Input.Properties != nil {
			goType = fmt.Sprintf("%s%sArgs", serviceTitle, methodName)
		} else {
			goType = gentypes.GoTypeFromSchema(action.Input)
		}
		argsString = fmt.Sprintf("senderID string, %s %s", argName, goType)
	}
	// output always returns an error
	respString := "error"
	if action.Output != nil {
		respName := getParamName("resp", action.Output)
		goType := gentypes.GoTypeFromSchema(action.Output)

		// if the output is an object with a schema then use schema as the type
		if action.Output.Type == "array" {
			// the type of an array is determined by its 'items' dataschema:
			//  when items is an object dataschema then the type is a predefined RespType
			itemsType := action.Output.ArrayItems
			if itemsType != nil && itemsType.Type == "object" && itemsType.Schema == "" {
				// this special array-of-objects case has to be handled here as the
				// type is already predefined in the Types file.
				respName = serviceTitle + methodName
				goType = fmt.Sprintf("%sResp", respName)
			} else {
				// otherwise use as-is
				goType = gentypes.GoTypeFromSchema(action.Output)
			}
		}

		//}
		respString = fmt.Sprintf("(%s %s, err error)", respName, goType)
	}
	l.Add("")
	l.Add("   // %s %s", methodName, action.Title)
	if action.Description != "" {
		l.Add("   // %s", action.Description)
	}
	if action.Output != nil && action.Output.Description != "" {
		desc := gentypes.FirstToLower(action.Output.Description)
		l.Add("   // This returns a %s", desc)
	}
	l.Add("   %s(%s) %s", methodName, argsString, respString)
}

// Generate a parameter name from the schema title.
// Parameter names start with lower case and consist only of alpha-num chars
// Intended to make the api more readable.
func getParamName(defaultName string, ds *td.DataSchema) string {
	if ds.Title == "" {
		return defaultName
	}
	str := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(ds.Title, "")
	str = gentypes.FirstToLower(str)
	return str
}
