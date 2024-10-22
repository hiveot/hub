package hubclient

import (
	"fmt"
)

// AgentHandler is a helper that maps messages to a Thing (service) invocation
// On receiving a message for a Thing:
//  1. looks-up the method name and obtains the registered method
//     2a. if the method has an argument (args struct) then
//     2.1 Instantiate the args struct
//     2.2 Decode the message request data into the arguments struct
//     2.3 invoke the method with the argument
//     2b. if the method doesn't have an argument
//     2.4 invoke the method without an argument
type AgentHandler struct {
	// the thing this agent is a handler for
	thingID string
	methods map[string]interface{}
}

func (agent *AgentHandler) InvokeMethod(
	method interface{}, msg *ThingMessage) (stat ActionProgress) {

	respData, err := HandleRequestMessage(msg.SenderID, method, msg.Data)
	stat.Completed(msg, respData, err)
	return stat
}

func (agent *AgentHandler) HandleMessage(msg *ThingMessage) (stat ActionProgress) {
	if msg.ThingID == agent.thingID {
		method, found := agent.methods[msg.Name]
		if found {
			return agent.InvokeMethod(method, msg)
		}
	}
	stat.Failed(msg, fmt.Errorf(
		"agent for service '%s' does not have method '%s'", agent.thingID, msg.Name))
	return stat
}

func NewAgentHandler(thingID string, methods map[string]interface{}) *AgentHandler {
	agent := AgentHandler{
		thingID: thingID,
		methods: methods,
	}
	return &agent
}
