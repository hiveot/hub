package _go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
)

// GenServiceHandler generates a function that returns a handler that unmarshal a request and invoke service
// The signature is: GetActionHandler(service I...Service) MessageHandler
func GenServiceHandler(l *utils.L, td *things.TD) {
	// ServiceType is the type of the service implementation that handles the messages
	serviceType := ToTitle(td.GetID()) + "Service"
	l.Add("")
	l.Add("// NewActionHandler returns a handler for Thing '%s' actions to be passed to the implementing service", td.ID)
	l.Add("//")
	l.Add("// This unmarshals the request payload into a args struct and passes it to the service")
	l.Add("// that implements the corresponding interface method.")
	l.Add("// ")
	l.Add("// This returns the marshalled response data or an error.")
	l.Add("func NewActionHandler(svc I%s)(func(*things.ThingMessage) api.DeliveryStatus) {", serviceType)
	l.Add("	return func(msg *things.ThingMessage) api.DeliveryStatus {")
	l.Add("		var err = fmt.Errorf(\"unknown action '%s'\",msg.Key)", "%s")
	l.Add(" 		var status = api.DeliveryFailed")
	l.Add("		res := api.DeliveryStatus{}")
	l.Add("		switch msg.Key {")
	for key, action := range td.Actions {
		GenActionHandler(l, serviceType, key, action)
	}
	l.Add("		}")
	l.Add("		res.Status = status")
	l.Add("		if err != nil {")
	l.Add("			res.Error = err.Error()")
	l.Add("		}")

	l.Add("		return res")
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
		if action.Output != nil {
			l.Add("				var resp interface{}")
		}
		l.Add("				err = json.Unmarshal(msg.Data, &args)")
		argsString = "args"
	}
	// build the result string, either an error or a response struct with an error
	resultString := "err"
	if action.Output != nil {
		resultString = "resp, err"
	}
	l.Add("				%s = svc.%s(%s)", resultString, methodName, argsString)
	if action.Output != nil {
		l.Add("				if err == nil {")
		l.Add("					res.Reply, err = json.Marshal(resp)")
		l.Add("					status = api.DeliveryCompleted")
		l.Add("				}")
		l.Add("				break")
	} else {
		l.Add("				if err == nil {")
		l.Add("					status = api.DeliveryCompleted")
		l.Add("				}")
		l.Add("				break")
	}
}
