package tdd2go

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
)

// GenServiceHandler generates a function that returns a handler that unmarshal a request and invoke service
// The signature is: New<ThingID>Handler(service I...Service) MessageHandler
func GenServiceHandler(l *utils.SL, serviceTitle string, td *tdd.TD) {
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
	l.Add("func New%sHandler(svc %s)(func(*hubclient.ThingMessage) hubclient.DeliveryStatus) {", serviceTitle, interfaceName)
	l.Indent++
	l.Add("return func(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {")

	l.Indent++
	l.Add("var err error")
	l.Add("var resp interface{}")
	l.Add("var senderID = msg.SenderID")

	l.Add("switch msg.Name {")
	l.Indent++
	for name, action := range td.Actions {
		GenActionHandler(l, serviceTitle, name, action)
	}
	l.Add("default:")
	l.Add("	err = errors.New(\"Unknown Method '\"+msg.Name+\"' of service '\"+msg.ThingID+\"'\")")
	l.Add("	stat.Failed(msg,err)")
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
// name  of the action affordance in the TD
func GenActionHandler(l *utils.SL, serviceTitle string, name string, action *tdd.ActionAffordance) {
	methodName := Key2ID(name)
	// build the argument string
	argsString := "senderID" // all handlers receive the sender ID
	l.Add("case \"%s\":", name)
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
		l.Add("err = utils.DecodeAsObject(msg.Data,&args)")

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
