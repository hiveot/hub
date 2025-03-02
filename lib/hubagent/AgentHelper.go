package hubagent

import (
	"fmt"
	"github.com/hiveot/hub/messaging"
)

// Deprecated: This is a reflection based request handler that invokes a service
// native API with the request parameters. Intended to eliminate boilerplate code.
// While clever it is hard to debug and trace and reflection isnt super fast.
// Replaced with the td2go generator that generates easy to read boilerplate code.
// Services are expected to define a TD and generate their API from the TD.
// This file is kept for a while to not lose the reflection magic.

// AgentHandler is a helper that maps messages to a Thing (service) invocation
//
// On receiving a message for a Thing:
//  1. looks-up the method name and obtains the registered method
//     2a. if the method has an argument (args struct) then
//     2.1 Instantiate the args struct
//     2.2 Decode the message request data into the arguments struct
//     2.3 invoke the method with the argument
//     2b. if the method doesn't have an argument
//     2.4 invoke the method without an argument
//
// Replace with generated clients using td2go, that doesnt use reflection.
type AgentHandler struct {
	// the thing this agent is a handler for
	thingID string
	methods map[string]interface{}
}

func (agent *AgentHandler) InvokeMethod(
	method interface{}, msg *messaging.RequestMessage) (output any, err error) {

	respData, err := HandleRequestMessage(msg.SenderID, method, msg.Input)
	return respData, err
}

func (agent *AgentHandler) HandleRequest(
	req *messaging.RequestMessage, _ messaging.IConnection) *messaging.ResponseMessage {

	if req.ThingID == agent.thingID {
		method, found := agent.methods[req.Name]
		if found {
			output, err := agent.InvokeMethod(method, req)
			return req.CreateResponse(output, err)
		}
	}
	err := fmt.Errorf(
		"agent for service '%s' does not have method '%s'", agent.thingID, req.Name)
	return req.CreateResponse(nil, err)
}

func NewAgentHandler(thingID string, methods map[string]interface{}) *AgentHandler {
	agent := AgentHandler{
		thingID: thingID,
		methods: methods,
	}
	return &agent
}
