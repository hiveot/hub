package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
)

// AuthnAgent agent for the authentication services:
type AuthnAgent struct {
	adminHandler transports.RequestHandler
	userHandler  transports.RequestHandler
}

// HandleAction authn services action request
func (agent *AuthnAgent) HandleAction(msg *transports.ThingMessage) (output any, err error) {

	_, thingID := td.SplitDigiTwinThingID(msg.ThingID)
	if thingID == authn.AdminServiceID {
		output, err = agent.adminHandler(msg)
	} else if thingID == authn.UserServiceID {
		output, err = agent.userHandler(msg)
	} else {
		err = fmt.Errorf("unknown authn service capability '%s'", msg.ThingID)
	}
	return output, err
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
