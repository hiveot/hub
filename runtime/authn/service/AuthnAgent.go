package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
)

// AuthnAgent agent for the authentication services:
type AuthnAgent struct {
	adminHandler api.ActionHandler
	userHandler  api.ActionHandler
}

// HandleAction authn services action request
func (agent *AuthnAgent) HandleAction(msg *transports.ThingMessage) (stat transports.RequestStatus) {

	_, thingID := tdd.SplitDigiTwinThingID(msg.ThingID)
	if thingID == authn.AdminServiceID {
		stat = agent.adminHandler(msg)
	} else if thingID == authn.UserServiceID {
		stat = agent.userHandler(msg)
	} else {
		err := fmt.Errorf("unknown authn service capability '%s'", msg.ThingID)
		stat.Failed(msg, err)
	}
	return stat
}

// StartAuthnAgent returns a new instance of the agent for the authentication services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the authentication service whose capabilities to expose
func StartAuthnAgent(svc *AuthnService) *AuthnAgent {
	agent := &AuthnAgent{
		adminHandler: authn.NewHandleAdminAction(svc.AdminSvc),
		userHandler:  authn.NewHandleUserAction(svc.UserSvc),
	}
	return agent
}
