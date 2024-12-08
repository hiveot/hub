package service

import (
	"fmt"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
)

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  transports.IClientConnection
	svc *StateService
}

// HandleRequest dispatches requests to the service capabilities
func (agent *StateAgent) HandleRequest(msg *transports.ThingMessage) (output any, err error) {
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
	err = fmt.Errorf("unknown action '%s' for service '%s'", msg.Name, msg.ThingID)
	return nil, err
}
func (agent *StateAgent) Delete(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.DeleteArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		err = agent.svc.Delete(msg.SenderID, args.Key)
	}
	return nil, err
}
func (agent *StateAgent) Get(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.GetArgs{}
	resp := stateapi.GetResp{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err != nil {
		resp.Key = args.Key
		resp.Value, resp.Found, err = agent.svc.Get(msg.SenderID, args.Key)
	}
	return resp, err
}
func (agent *StateAgent) GetMultiple(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.GetMultipleArgs{}
	resp := stateapi.GetMultipleResp{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		resp.KV, err = agent.svc.GetMultiple(msg.SenderID, args.Keys)
	}
	return resp, err
}
func (agent *StateAgent) Set(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.SetArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err != nil {
		err = agent.svc.Set(msg.SenderID, args.Key, args.Value)
	}
	return nil, err
}
func (agent *StateAgent) SetMultiple(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.SetMultipleArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		err = agent.svc.SetMultiple(msg.SenderID, args.KV)
	}
	return nil, err
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
func StartStateAgent(svc *StateService, hc transports.IClientConnection) *StateAgent {
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		hc.SetRequestHandler(agent.HandleRequest)
	}
	return &agent
}
