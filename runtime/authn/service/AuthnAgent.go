package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
)

// AuthnAgent agent for the authentication services:
type AuthnAgent struct {
	hc hubclient.IHubClient
	//hc           *hubclient.HubClient
	svc *AuthnService

	adminHandler hubclient.MessageHandler
	userHandler  hubclient.MessageHandler
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *AuthnAgent) HandleMessage(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	// if the message has an authn agent prefix then remove it.
	// This can happen if invoked directly through an embedded client
	_, thingID := tdd.SplitDigiTwinThingID(msg.ThingID)
	if thingID == authn.AdminServiceID {
		return agent.adminHandler(msg)
	} else if thingID == authn.UserServiceID {
		return agent.userHandler(msg)
	}
	err := fmt.Errorf("unknown authn service capability '%s'", msg.ThingID)
	stat.Failed(msg, err)
	return stat
}

// StartAuthnAgent returns a new instance of the agent for the authentication services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the authentication service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartAuthnAgent(
	svc *AuthnService, hc hubclient.IHubClient) (*AuthnAgent, error) {
	var err error

	agent := AuthnAgent{hc: hc, svc: svc}
	agent.adminHandler = authn.NewAdminHandler(svc.AdminSvc)
	agent.userHandler = authn.NewUserHandler(svc.UserSvc)
	if hc != nil {
		agent.hc.SetMessageHandler(agent.HandleMessage)
	}
	return &agent, err
}
