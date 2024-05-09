package authnagent

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
)

// AuthnAgentID is the connection ID of the authn agent used in providing its capabilities
const AuthnAgentID = "authn"

// AuthnAgent agent for the authentication services:
type AuthnAgent struct {
	tp transports.IHubTransport
	//hc           *hubclient.HubClient
	svc *service.AuthnService

	adminHandler api.MessageHandler
	userHandler  api.MessageHandler
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *AuthnAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.ThingID == api.AuthnAdminThingID {
		return agent.adminHandler(msg)
	} else if msg.ThingID == api.AuthnUserThingID {
		return agent.userHandler(msg)
	}
	stat.Error = fmt.Sprintf("unknown authn service capability '%s'", msg.ThingID)
	stat.Status = api.DeliveryFailed
	return stat
}

// StartAuthnAgent returns a new instance of the agent for the authentication service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the authorization service whose capabilities to expose
//	tp is the optional message client connected to the server protocol
func StartAuthnAgent(
	svc *service.AuthnService, tp transports.IHubTransport) (*AuthnAgent, error) {
	var err error
	agent := AuthnAgent{tp: tp, svc: svc}
	agent.adminHandler = NewAuthnAdminHandler(agent.svc.AdminSvc)
	agent.userHandler = NewAuthnUserHandler(agent.svc.UserSvc)
	if tp != nil {
		agent.tp.SetMessageHandler(agent.HandleMessage)
		// agents don't need to subscribe for actions directed at them
		//err = agent.tp.Subscribe(api.AuthnAdminThingID)
		//if err == nil {
		//	err = agent.tp.Subscribe(api.AuthnUserThingID)
		//}
	}
	return &agent, err
}
