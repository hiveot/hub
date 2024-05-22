package authzagent

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authz"
)

// AuthzAgentID is the agent's connection ID used addressing its capabilities
const AuthzAgentID = "authz"

// AuthzAgent serves the message based interface to the authz service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthzAgent struct {
	hc  hubclient.IHubClient
	svc *authz.AuthzService
}

// HandleMessage an event or action message for the authz service
// This message is send by the protocol client connected to this agent
func (agent *AuthzAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.ThingID == api.AuthzThingID {
		if msg.MessageType == vocab.MessageTypeAction {
			switch msg.Key {
			case api.GetClientRoleMethod:
				return agent.GetClientRole(msg)
			case api.SetClientRoleMethod:
				return agent.SetClientRole(msg)
			}
		}
		stat.Error = fmt.Sprintf("unknown authz action '%s' for capability '%s'", msg.Key, msg.ThingID)
	} else {
		stat.Error = fmt.Sprintf("unknown authz service capability '%s'", msg.ThingID)
	}
	stat.Status = api.DeliveryFailed
	return stat
}

func (agent *AuthzAgent) GetClientRole(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	var args api.GetClientRoleArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		stat.Status = api.DeliveryCompleted
		role, err2 := agent.svc.GetClientRole(args.ClientID)
		err = err2
		if err == nil {
			resp := api.GetClientRoleResp{ClientID: args.ClientID, Role: role}
			stat.Reply, err = json.Marshal(resp)
		}
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	}
	return stat
}

func (agent *AuthzAgent) SetClientRole(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {
	var args api.SetClientRoleArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		stat.Status = api.DeliveryCompleted
		err = agent.svc.SetClientRole(args.ClientID, args.Role)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	}
	return stat
}

// StartAuthzAgent creates a new instance of the agent handling authorization service requests
// If hc is nil then use the HandleMessage method directly to pass messages to the agent,
// for example when testing.
//
//	svc is the authorization service whose capabilities to expose
//	hc is the optional message client used to publish and subscribe
func StartAuthzAgent(svc *authz.AuthzService, hc hubclient.IHubClient) (*AuthzAgent, error) {
	var err error
	agent := AuthzAgent{svc: svc, hc: hc}
	if hc != nil {
		agent.hc.SetActionHandler(agent.HandleMessage)
		// agents don't need to subscribe to receive actions directed at them
		//err = agent.hc.Subscribe(api.AuthzThingID)
	}
	return &agent, err
}
