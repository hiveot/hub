package _go

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenServiceInterface generates the service interface for handling messages
func GenServiceInterface(l *utils.L, serviceName string, td *things.TD) {
	// ServiceType is the interface of the service. Interface names start with 'I'
	interfaceName := "I" + serviceName + "Service"
	l.Add("")
	l.Add("// %s defines the interface of the '%s' service", interfaceName, serviceName)
	l.Add("//")
	l.Add("// This defines a method for each of the actions in the TD. ")
	l.Add("// ")
	l.Add("type %s interface {", interfaceName)
	for key, action := range td.Actions {
		GenInterfaceMethod(l, key, action)
	}
	l.Add("}")
}

// GenInterfaceMethod adds a method definition for an action
// service methods receive a request struct and return a reply struct, if defined
func GenInterfaceMethod(l *utils.L, key string, action *things.ActionAffordance) {
	//attrs := GetSchemaAttrs("arg", action.Input)
	methodName := Key2ID(key)
	argsString := ""
	if action.Input != nil {
		argsString = fmt.Sprintf("args %sArgs", methodName)
	}
	respString := "error"
	if action.Output != nil {
		respString = fmt.Sprintf("(%sResp, error)", methodName)
	}
	l.Add("")
	l.Add("   // %s %s", methodName, action.Title)
	if action.Description != "" {
		l.Add("   // %s", action.Description)
	}
	l.Add("   %s(%s) %s", methodName, argsString, respString)
}
