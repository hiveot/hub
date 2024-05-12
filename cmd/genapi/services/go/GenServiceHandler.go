package _go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenServiceHandler generates a function that returns a handler that unmarshal a request and invoke service
// The signature is: GetActionHandler(service I...Service) MessageHandler
func GenServiceHandler(l *utils.L, serviceName string, td *things.TD) {
	// ServiceType is the type of the service implementation that handles the messages
	//serviceType := ToTitle(td.GetID()) + "Service"
	interfaceName := "I" + serviceName + "Service"

	l.Add("")
	l.Add("// NewActionHandler returns a server handler for Thing '%s' actions.", td.ID)
	l.Add("//")
	l.Add("// This unmarshals the request payload into an args struct and passes it to the service")
	l.Add("// that implements the corresponding interface method.")
	l.Add("// ")
	l.Add("// This returns the marshalled response data or an error.")
	l.Add("func NewActionHandler(svc %s)(func(*things.ThingMessage) api.DeliveryStatus) {", interfaceName)
	l.Add("	return func(msg *things.ThingMessage) (stat api.DeliveryStatus) {")

	l.Add(" 		var err error")
	l.Add("		switch msg.Key {")
	for key, action := range td.Actions {
		GenActionHandler(l, serviceName, key, action)
	}
	l.Add("		default:")
	l.Add("			err = errors.New(\"Unknown Method '\"+msg.Key+\"' of service '\"+msg.ThingID+\"'\")")
	l.Add("			stat.Failed(msg, err)")
	l.Add("		}")

	l.Add("		return stat")
	l.Add("	}")
	l.Add("}")
}

// GenActionHandler add a method handler for an action.
// This unmarshal the request, invokes the service, and marshals the response
// key is the key of the action affordance in the TD
func GenActionHandler(l *utils.L, serviceType string, key string, action *things.ActionAffordance) {
	methodName := Key2ID(key)
	// build the argument string
	argsString := ""
	l.Add("			case \"%s\":", key)
	if action.Input != nil {
		l.Add("				args := %sArgs{}", methodName)
		l.Add("				err = json.Unmarshal(msg.Data, &args)")
		argsString = "args"
	}
	// build the result string, either an error or a response struct with an error
	resultString := "err"
	if action.Output != nil {
		l.Add("				var resp interface{}")
		resultString = "resp, err"
	}
	l.Add("				if err == nil {")
	l.Add("					%s = svc.%s(%s)", resultString, methodName, argsString)
	l.Add("				}")
	if action.Output != nil {
		l.Add("				if resp != nil {")
		l.Add("					stat.Reply, _ = json.Marshal(resp)")
		l.Add("				}")
		l.Add("				stat.Completed(msg, err)")
		l.Add("				break")
	} else {
		l.Add("				break")
	}
}
