package tdd2go

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
)

// GenServiceInterface generates the service interface for handling messages
func GenServiceInterface(l *utils.SL, serviceTitle string, td *tdd.TD) {
	// ServiceType is the interface of the service. Interface names start with 'I'
	interfaceName := "I" + serviceTitle + "Service"
	l.Add("")
	l.Add("// %s defines the interface of the '%s' service", interfaceName, serviceTitle)
	l.Add("//")
	l.Add("// This defines a method for each of the actions in the TD. ")
	l.Add("// ")
	l.Add("type %s interface {", interfaceName)

	sortedActionNames := utils.OrderedMapKeys(td.Actions)
	for _, name := range sortedActionNames {
		action := td.Actions[name]
		GenInterfaceMethod(l, serviceTitle, name, action)
	}
	l.Add("}")
}

// GenInterfaceMethod adds a method definition for an action.
// The generated method arguments are the senderID and the value.
// The value is either a native type or a struct, based on the TDD definition
func GenInterfaceMethod(l *utils.SL, serviceTitle string, name string, action *tdd.ActionAffordance) {
	//attrs := GetSchemaAttrs("arg", action.Input)
	methodName := Key2ID(name)
	argsString := "senderID string" // all methods receive the sender ID
	if action.Input != nil {
		argName := getParamName("args", action.Input)
		goType := ""
		if action.Input.Type == "object" && action.Input.Properties != nil {
			goType = fmt.Sprintf("%s%sArgs", serviceTitle, methodName)
		} else {
			goType = GoTypeFromSchema(action.Input)
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
			goType = GoTypeFromSchema(action.Output)
		}
		respString = fmt.Sprintf("(%s %s, err error)", respName, goType)
	}
	l.Add("")
	l.Add("   // %s %s", methodName, action.Title)
	if action.Description != "" {
		l.Add("   // %s", action.Description)
	}
	if action.Output != nil && action.Output.Description != "" {
		desc := FirstToLower(action.Output.Description)
		l.Add("   // This returns a %s", desc)
	}
	l.Add("   %s(%s) %s", methodName, argsString, respString)
}
