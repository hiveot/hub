package tdd2go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenServiceHandler generates a function that returns a handler that unmarshal a request and invoke service
// The signature is: GetActionHandler(service I...Service) MessageHandler
func GenServiceHandler(l *utils.L, serviceTitle string, td *things.TD) {
	// ServiceType is the type of the service implementation that handles the messages
	//serviceType := ToTitle(td.GetID()) + "Service"
	interfaceName := "I" + serviceTitle + "Service"

	l.Add("")
	l.Add("// New%sHandler returns a server handler for Thing '%s' actions.", serviceTitle, td.ID)
	l.Add("//")
	l.Add("// This unmarshalls the request payload into an args struct and passes it to the service")
	l.Add("// that implements the corresponding interface method.")
	l.Add("// ")
	l.Add("// This returns the marshalled response data or an error.")
	l.Add("func New%sHandler(svc %s)(func(*things.ThingMessage) hubclient.DeliveryStatus) {", serviceTitle, interfaceName)
	l.Indent++
	l.Add("return func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {")

	l.Indent++
	l.Add("var err error")
	l.Add("var resp interface{}")
	l.Add("var senderID = msg.SenderID")

	l.Add("switch msg.Key {")
	l.Indent++
	for key, action := range td.Actions {
		GenActionHandler(l, serviceTitle, key, action)
	}
	l.Add("default:")
	l.Add("	err = errors.New(\"Unknown Method '\"+msg.Key+\"' of service '\"+msg.ThingID+\"'\")")
	l.Add("	stat.DeliveryFailed(msg,err)")
	l.Indent--
	l.Add("}")

	l.Add("stat.Completed(msg, resp, err)")
	l.Add("return stat")
	l.Indent--
	l.Add("}")
	l.Indent--
	l.Add("}")
}

// GenActionHandler add an unmarshaller handler for its service.
// This unmarshal the request, invokes the service, and marshals the response
// key is the key of the action affordance in the TD
func GenActionHandler(l *utils.L, serviceTitle string, key string, action *things.ActionAffordance) {
	methodName := Key2ID(key)
	// build the argument string
	argsString := "senderID" // all handlers receive the sender ID
	l.Add("case \"%s\":", key)
	l.Indent++
	if action.Input != nil {
		if action.Input.Type == "object" && action.Input.Properties != nil {
			// objects are passed by their struct type
			l.Add("args := %s%sArgs{}", serviceTitle, methodName)
		} else {
			// native types are passed as-is
			goType := GoTypeFromSchema(action.Input)
			l.Add("var args %s", goType)
		}
		l.Add("err = msg.Decode(&args)")
		argsString += ", args"
	}
	// build the result string, either an error or a response struct with an error
	resultString := "err"
	if action.Output != nil {
		resultString = "resp, err"
	}
	l.Add("if err == nil {")
	l.Add("  %s = svc.%s(%s)", resultString, methodName, argsString)
	l.Add("} else {")
	l.Add("  err = errors.New(\"bad function argument: \"+err.Error())")
	l.Add("}")
	l.Add("break")
	l.Indent--
}
