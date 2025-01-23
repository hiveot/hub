package td2go

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
)

// GenServiceHandler generates a function that returns an action handler that
// unmarshals a request and invoke the requested service.
//
// The signature is: NewHandle<ThingID>Action(service I...Service) *HandleRequest
//
//	func (agent *...)NewHandleAction(
//	  consumerID string, dThingID string, actionName string, input any, correlationID string
func GenServiceHandler(l *utils.SL, serviceTitle string, td1 *td.TD) {
	// ServiceType is the type of the service implementation that handles the messages
	//serviceType := ToTitle(td1.GetID()) + "Service"
	interfaceName := "I" + serviceTitle + "Service"

	l.Add("")
	l.Add("// NewHandle%sRequest returns an agent handler for Thing '%s' requests.", serviceTitle, td1.ID)
	l.Add("//")
	l.Add("// This unmarshalls the request payload into an args struct and passes it to the service")
	l.Add("// that implements the corresponding interface method.")
	l.Add("// ")
	l.Add("// This returns the marshalled response data or an error.")
	l.Add("func NewHandle%sRequest(svc %s)(func(msg *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage) {",
		serviceTitle, interfaceName)
	l.Indent++
	l.Add("return func(msg *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {")

	l.Indent++

	l.Add("var output any")
	l.Add("var err error")
	l.Add("switch msg.Name {")
	l.Indent++
	for name, action := range td1.Actions {
		GenActionHandler(l, serviceTitle, name, action)
	}
	l.Add("default:")
	l.Add("	err = errors.New(\"Unknown Method '\"+msg.Name+\"' of service '\"+msg.ThingID+\"'\")")
	l.Indent--
	l.Add("}")

	l.Add("return msg.CreateResponse(output,err)")
	l.Indent--
	l.Add("}")
	l.Indent--
	l.Add("}")
}

// GenActionHandler add an unmarshaller handler for its service.
// This unmarshal the request, invokes the service, and marshals the response
// name  of the action affordance in the TD
func GenActionHandler(l *utils.SL, serviceTitle string, name string, action *td.ActionAffordance) {
	methodName := Name2ID(name)
	// build the argument string
	argsString := "msg.SenderID" // all handlers receive the sender ID
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
		l.Add("err = tputils.DecodeAsObject(msg.Input, &args)")

		argsString += ", args"
	}
	// build the result string, either an error or a response struct with an error
	resultString := "err"
	if action.Output != nil {
		resultString = "output, err"
	}
	l.Add("if err == nil {")
	l.Add("  %s = svc.%s(%s)", resultString, methodName, argsString)
	l.Add("} else {")
	l.Add("  err = errors.New(\"bad function argument: \"+err.Error())")
	l.Add("}")
	l.Add("break")
	l.Indent--
}
