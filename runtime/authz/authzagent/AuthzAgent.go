package authzagent

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authz/service"
)

// AuthzAgent serves the message based interface to the authz service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthzAgent struct {
	hc  hubclient.IHubClient
	svc *service.AuthzService
}

// HandleMessage an event or action message for the authz service
// This message is send by the protocol client connected to this agent
func (agent *AuthzAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var err error
	// if the message has an authn agent prefix then remove it.
	// This can happen if invoked directly through an embedded client
	_, thingID := things.SplitDigiTwinThingID(msg.ThingID)
	if thingID == api.AuthzManageServiceID {
		if msg.MessageType == vocab.MessageTypeAction {
			switch msg.Key {
			case api.GetClientRoleMethod:
				return agent.GetClientRole(msg)
			case api.SetClientRoleMethod:
				return agent.SetClientRole(msg)
			}
		}
		err = fmt.Errorf("Authz: unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
	} else if thingID == api.AuthzUserServiceID {
		if msg.MessageType == vocab.MessageTypeAction {
			switch msg.Key {
			case api.SetPermissionsMethod:
				return agent.SetPermissions(msg)
			}
		}
		err = fmt.Errorf("Authz: unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
	} else {
		err = fmt.Errorf("Authz: unknown service '%s'", msg.ThingID)
	}
	stat.Completed(msg, err)
	return stat
}

func (agent *AuthzAgent) GetClientRole(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.GetClientRoleArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		role, err2 := agent.svc.GetClientRole(args.ClientID)
		err = err2
		if err == nil {
			resp := api.GetClientRoleResp{ClientID: args.ClientID, Role: role}
			stat.Reply, err = json.Marshal(resp)
		}
	}
	stat.Completed(msg, err)
	return stat
}

func (agent *AuthzAgent) SetClientRole(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var args api.SetClientRoleArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = agent.svc.SetClientRole(args.ClientID, args.Role)
	}
	stat.Completed(msg, err)
	return stat
}
func (agent *AuthzAgent) SetPermissions(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var args api.ThingPermissions
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = agent.svc.SetPermissions(msg.SenderID, args)
	}
	stat.Completed(msg, err)
	return stat
}

// StartAuthzAgent creates a new instance of the agent handling authorization service requests
// If hc is nil then use the HandleMessage method directly to pass messages to the agent,
// for example when testing.
//
//	svc is the authorization service whose capabilities to expose
//	hc is the optional message client used to publish and subscribe
func StartAuthzAgent(svc *service.AuthzService, hc hubclient.IHubClient) (*AuthzAgent, error) {
	var err error
	agent := AuthzAgent{svc: svc, hc: hc}
	if hc != nil {
		agent.hc.SetActionHandler(agent.HandleMessage)
		// agents don't need to subscribe to receive actions directed at them
		//err = agent.hc.Subscribe(api.AuthzThingID)
	}
	return &agent, err
}
