package genagent

import (
	"fmt"
	"github.com/hiveot/hub/cmd/td2go/gentypes"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
	"regexp"
)

// GenServiceInterface generates the interface the service has to implement.
//
//	agentID is this package name, eg: the agent for this service
//	serviceID is the ThingID of the service capitalized
func GenServiceInterface(l *utils.SL, agentID, serviceID string, tdi *td.TD) {
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
// The generated method arguments are the senderID and the value.
// The value is either a native type or a struct, based on the TDD definition
func GenInterfaceMethod(l *utils.SL, serviceTitle string, name string, action *td.ActionAffordance) {

	// 1. build the input arguments. All methods receive the sender ID.
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
	respString := "error"
	if action.Output != nil {
		respName := getParamName("resp", action.Output)
		goType := ""
		if action.Output.Type == "object" && action.Output.Properties != nil {
			goType = fmt.Sprintf("%s%sResp", serviceTitle, methodName)
		} else {
			goType = gentypes.GoTypeFromSchema(action.Output)
		}
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
