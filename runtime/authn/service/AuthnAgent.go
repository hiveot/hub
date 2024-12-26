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

// HandleRequest authn services action request
func (agent *AuthnAgent) HandleRequest(
	req transports.RequestMessage) (resp transports.ResponseMessage) {

	_, thingID := td.SplitDigiTwinThingID(req.ThingID)
	if thingID == authn.AdminServiceID {
		resp = agent.adminHandler(req)
	} else if thingID == authn.UserServiceID {
		resp = agent.userHandler(req)
	} else {
		err := fmt.Errorf("unknown authn service capability '%s'", req.ThingID)
		resp = req.CreateResponse(nil, err)
	}
	return resp
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
