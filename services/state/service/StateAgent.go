package service

import (
	"fmt"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/wot/transports/utils"
)

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  clients.IAgent
	svc *StateService
}

// HandleRequest dispatches requests to the service capabilities
func (agent *StateAgent) HandleRequest(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	if msg.ThingID == stateapi.StorageServiceID {
		switch msg.Name {
		case stateapi.DeleteMethod:
			return agent.Delete(msg)
		case stateapi.GetMethod:
			return agent.Get(msg)
		case stateapi.GetMultipleMethod:
			return agent.GetMultiple(msg)
		case stateapi.SetMethod:
			return agent.Set(msg)
		case stateapi.SetMultipleMethod:
			return agent.SetMultiple(msg)
		}
	}
	stat.Failed(msg, fmt.Errorf(
		"unknown action '%s' for service '%s'", msg.Name, msg.ThingID))
	return stat
}
func (agent *StateAgent) Delete(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	args := stateapi.DeleteArgs{}
	err := utils.DecodeAsObject(msg.Data, &args)
	err = agent.svc.Delete(msg.SenderID, args.Key)
	stat.Completed(msg, nil, err)
	return stat
}
func (agent *StateAgent) Get(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	args := stateapi.GetArgs{}
	resp := stateapi.GetResp{}
	err := utils.DecodeAsObject(msg.Data, &args)
	resp.Key = args.Key
	resp.Value, resp.Found, err = agent.svc.Get(msg.SenderID, args.Key)
	stat.Completed(msg, resp, err)
	return stat
}
func (agent *StateAgent) GetMultiple(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	args := stateapi.GetMultipleArgs{}
	resp := stateapi.GetMultipleResp{}
	err := utils.DecodeAsObject(msg.Data, &args)
	resp.KV, err = agent.svc.GetMultiple(msg.SenderID, args.Keys)
	stat.Completed(msg, resp, err)
	return stat
}
func (agent *StateAgent) Set(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	args := stateapi.SetArgs{}
	err := utils.DecodeAsObject(msg.Data, &args)
	err = agent.svc.Set(msg.SenderID, args.Key, args.Value)
	stat.Completed(msg, nil, err)
	return stat
}
func (agent *StateAgent) SetMultiple(msg *transports.ThingMessage) (stat transports.RequestStatus) {
	args := stateapi.SetMultipleArgs{}
	err := utils.DecodeAsObject(msg.Data, &args)
	err = agent.svc.SetMultiple(msg.SenderID, args.KV)
	stat.Completed(msg, nil, err)
	return stat
}

// NewStateAgent returns a new instance of the communication agent for the state service.
// Intended for use in StartPlugin
//
//	svc is the state service whose capabilities to expose
func NewStateAgent(svc *StateService) *StateAgent {
	agent := StateAgent{svc: svc}
	return &agent
}

// StartStateAgent returns a new instance of the communication agent for the state service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
//
//	svc is the state service whose capabilities to expose
//	hc is the messaging client used to register a message handler
func StartStateAgent(svc *StateService, hc clients.IAgent) *StateAgent {
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		hc.SetRequestHandler(agent.HandleRequest)
	}
	return &agent
}
