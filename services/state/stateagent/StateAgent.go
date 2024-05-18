package stateagent

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/services/state/service"
	"github.com/hiveot/hub/services/state/stateapi"
)

// StateAgentID is the connection ID of the state agent used in providing its capabilities
const StateAgentID = "state"

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  hubclient.IHubClient
	svc *service.StateService
}

// HandleMessage dispatches requests to the service capabilities
func (agent *StateAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.ThingID == stateapi.StorageThingID {
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
	stat.Failed(msg, fmt.Errorf(
		"unknown action '%s' for service '%s'", msg.Key, msg.ThingID))
	return stat
}
func (agent *StateAgent) Delete(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	args := stateapi.DeleteArgs{}
	err := json.Unmarshal(msg.Data, &args)
	err = agent.svc.Delete(msg.SenderID, args.Key)
	stat.Completed(msg, err)
	return stat
}
func (agent *StateAgent) Get(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	args := stateapi.GetArgs{}
	resp := stateapi.GetResp{}
	err := json.Unmarshal(msg.Data, &args)
	resp.Key = args.Key
	resp.Value, err = agent.svc.Get(msg.SenderID, args.Key)
	stat.Completed(msg, err)
	resp.Found = (err == nil)
	stat.Reply, _ = json.Marshal(resp)
	return stat
}
func (agent *StateAgent) GetMultiple(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	args := stateapi.GetMultipleArgs{}
	resp := stateapi.GetMultipleResp{}
	err := json.Unmarshal(msg.Data, &args)
	resp.KV, err = agent.svc.GetMultiple(msg.SenderID, args.Keys)
	stat.Completed(msg, err)
	stat.Reply, _ = json.Marshal(resp)
	return stat
}
func (agent *StateAgent) Set(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	args := stateapi.SetArgs{}
	err := json.Unmarshal(msg.Data, &args)
	err = agent.svc.Set(msg.SenderID, args.Key, args.Value)
	stat.Completed(msg, err)
	return stat
}
func (agent *StateAgent) SetMultiple(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	args := stateapi.SetMultipleArgs{}
	err := json.Unmarshal(msg.Data, &args)
	err = agent.svc.SetMultiple(msg.SenderID, args.KV)
	stat.Completed(msg, err)
	return stat
}

// StartStateAgent returns a new instance of the communication agent for the state service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
//
//	svc is the state service whose capabilities to expose
//	hc is the messaging client used to register a message handler
func StartStateAgent(
	svc *service.StateService, hc hubclient.IHubClient) (*StateAgent, error) {
	var err error
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		agent.hc.SetMessageHandler(agent.HandleMessage)
	}
	// FIXME: REINSTATE AUTHORIZATION FOR SERVICES
	// Set the required permissions for using this service
	// any user roles can read and write their state
	//serviceProfile := authnclient.NewAuthnUserClient(svc.hc)
	//err = serviceProfile.SetServicePermissions(stateapi.StorageCap, []string{
	//	api.ClientRoleViewer,
	//	api.ClientRoleOperator,
	//	api.ClientRoleManager,
	//	api.ClientRoleAdmin,
	//	api.ClientRoleAgent,
	//	api.ClientRoleService})
	//if err != nil {
	//	return err
	//}
	return &agent, err
}
