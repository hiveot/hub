package _go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenServiceHandler generates a handler that unmarshal a request and invoke service
func GenServiceHandler(l *utils.L, td *things.TD) {
	// ServiceType is the type of the service implementation that handles the messages
	serviceType := ToTitle(td.GetID()) + "Service"
	l.Add("")
	l.Add("// HandleMessage handles messages for Thing '%s' to be passed to the implementing service", td.ID)
	l.Add("//")
	l.Add("// This unmarshals the request payload into a args struct and passes it to the service")
	l.Add("// that implements the corresponding interface method.")
	l.Add("// ")
	l.Add("// This returns the marshalled response data or an error.")
	l.Add("func HandleMessage(msg *things.ThingMessage, svc I%s)(reply []byte,err error) {", serviceType)
	l.Add("   switch msg.Key {")
	for key, action := range td.Actions {
		GenActionHandler(l, serviceType, key, action)
	}
	l.Add("   }")
	l.Add("   return nil, fmt.Errorf(\"unknown request method '%s'\",msg.Key)", "%s")
	l.Add("}")
}

// GenActionHandler add a method handler for an action.
// This unmarshal the request, invokes the service, and marshals the response
// key is the key of the action affordance in the TD
func GenActionHandler(l *utils.L, serviceType string, key string, action *things.ActionAffordance) {
	methodName := Key2ID(key)
	argsString := ""
	l.Add("   case \"%s\":", key)
	if action.Input != nil {
		l.Add("       args := %sArgs{}", methodName)
		l.Add("       err = json.Unmarshal(msg.Data, &args)")
		argsString = "args"
	}
	resultString := "err"
	if action.Output != nil {
		resultString = "resp, err"
	}
	l.Add("       %s := svc.%s(%s)", resultString, methodName, argsString)
	if action.Output != nil {
		l.Add("       reply,err = json.Marshal(resp)")
		l.Add("       return reply, err")
	} else {
		l.Add("       return nil, err")
	}
}
