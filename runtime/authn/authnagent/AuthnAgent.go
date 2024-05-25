package authnagent

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/service"
)

// AuthnAgent agent for the authentication services:
type AuthnAgent struct {
	hc hubclient.IHubClient
	//hc           *hubclient.HubClient
	svc *service.AuthnService

	adminHandler api.MessageHandler
	userHandler  api.MessageHandler
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *AuthnAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	// if the message has a authn agent prefix then remove it.
	// This can happen if invoked directly through an embedded client
	_, thingID := things.SplitDigiTwinThingID(msg.ThingID)
	if thingID == api.AuthnAdminServiceID {
		return agent.adminHandler(msg)
	} else if thingID == api.AuthnUserServiceID {
		return agent.userHandler(msg)
	}
	err := fmt.Errorf("unknown authn service capability '%s'", msg.ThingID)
	stat.Failed(msg, err)
	return stat
}

// StartAuthnAgent returns a new instance of the agent for the authentication service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the authentication service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartAuthnAgent(
	svc *service.AuthnService, hc hubclient.IHubClient) (*AuthnAgent, error) {
	var err error
	agent := AuthnAgent{hc: hc, svc: svc}
	agent.adminHandler = NewAuthnAdminHandler(agent.svc.AdminSvc)
	agent.userHandler = NewAuthnUserHandler(agent.svc.UserSvc)
	if hc != nil {
		agent.hc.SetActionHandler(agent.HandleMessage)
	}
	return &agent, err
}
