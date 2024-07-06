package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/state/stateapi"
)

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  hubclient.IHubClient
	svc *StateService
}

// HandleMessage dispatches requests to the service capabilities
func (agent *StateAgent) HandleMessage(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	if msg.ThingID == stateapi.StorageServiceID {
		switch msg.Key {
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
	stat.DeliveryFailed(msg, fmt.Errorf(
		"unknown action '%s' for service '%s'", msg.Key, msg.ThingID))
	return stat
}
func (agent *StateAgent) Delete(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	args := stateapi.DeleteArgs{}
	err := msg.Decode(&args)
	err = agent.svc.Delete(msg.SenderID, args.Key)
	stat.Completed(msg, nil, err)
	return stat
}
func (agent *StateAgent) Get(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	args := stateapi.GetArgs{}
	resp := stateapi.GetResp{}
	err := msg.Decode(&args)
	resp.Key = args.Key
	resp.Value, err = agent.svc.Get(msg.SenderID, args.Key)
	resp.Found = (err == nil)
	stat.Completed(msg, resp, err)
	return stat
}
func (agent *StateAgent) GetMultiple(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	args := stateapi.GetMultipleArgs{}
	resp := stateapi.GetMultipleResp{}
	err := msg.Decode(&args)
	resp.KV, err = agent.svc.GetMultiple(msg.SenderID, args.Keys)
	stat.Completed(msg, resp, err)
	return stat
}
func (agent *StateAgent) Set(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	args := stateapi.SetArgs{}
	err := msg.Decode(&args)
	err = agent.svc.Set(msg.SenderID, args.Key, args.Value)
	stat.Completed(msg, nil, err)
	return stat
}
func (agent *StateAgent) SetMultiple(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	args := stateapi.SetMultipleArgs{}
	err := msg.Decode(&args)
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
func StartStateAgent(svc *StateService, hc hubclient.IHubClient) *StateAgent {
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		agent.hc.SetMessageHandler(agent.HandleMessage)
	}
	return &agent
}
