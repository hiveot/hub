package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// AuthzAgent serves the message based interface to the authz service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthzAgent struct {
	hc           hubclient.IConsumerClient
	svc          *AuthzService
	adminHandler api.ActionHandler
	userHandler  api.ActionHandler
}

// HandleAction authz service action handler
func (agent *AuthzAgent) HandleAction(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {

	// if the message has an authn agent prefix then remove it.
	// This can happen if invoked directly through an embedded client
	_, thingID := tdd.SplitDigiTwinThingID(msg.ThingID)
	if thingID == authz.AdminServiceID {
		stat = agent.adminHandler(msg)
	} else if thingID == authz.UserServiceID {
		stat = agent.userHandler(msg)
	} else {
		err := fmt.Errorf("unknown authz service capability '%s'", msg.ThingID)
		stat.Failed(msg, err)
	}
	return stat
}

// HasPermission is a convenience function to check if the sender has permission
// to pub/sub events, actions or properties. This invokes HasPermission on the service.
func (agent *AuthzAgent) HasPermission(
	senderID, operation, dThingID string) bool {
	return agent.svc.HasPermission(senderID, operation, dThingID)
}

// StartAuthzAgent creates a new instance of the agent handling authorization service requests
// If hc is nil then use the HandleMessage method directly to pass messages to the agent,
// for example when testing.
//
//	svc is the authorization service whose capabilities to expose
//	hc is the optional message client used to publish and subscribe
func StartAuthzAgent(svc *AuthzService) (*AuthzAgent, error) {
	var err error
	agent := AuthzAgent{
		adminHandler: authz.NewHandleAdminAction(svc),
		userHandler:  authz.NewHandleUserAction(svc),
		svc:          svc,
	}

	// FIXME: replace authz with TD based permissions
	// set permissions for using the authn services as authz wasn't yet running
	err = svc.SetPermissions(authn.AdminAgentID, authz.ThingPermissions{
		AgentID: authn.AdminAgentID,
		ThingID: authn.AdminServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleService, authz.ClientRoleAdmin, authz.ClientRoleManager},
	})
	if err == nil {
		// all users with a role can GetProfile and refresh their token
		err = svc.SetPermissions(authn.UserAgentID, authz.ThingPermissions{
			AgentID: authn.UserAgentID,
			ThingID: authn.UserServiceID,
			Deny:    []authz.ClientRole{authz.ClientRoleNone, ""},
		})
	}

	// set permissions for using the authz services
	if err == nil {
		err = svc.SetPermissions(authz.AdminAgentID, authz.ThingPermissions{
			AgentID: authz.AdminAgentID,
			ThingID: authz.AdminServiceID,
			Allow:   []authz.ClientRole{authz.ClientRoleService, authz.ClientRoleAdmin, authz.ClientRoleManager},
		})
	}
	if err == nil {
		err = svc.SetPermissions(authz.UserAgentID, authz.ThingPermissions{
			AgentID: authz.UserAgentID,
			ThingID: authz.UserServiceID,
			Allow: []authz.ClientRole{authz.ClientRoleAgent, authz.ClientRoleService,
				authz.ClientRoleAdmin, authz.ClientRoleManager, authz.ClientRoleOperator},
		})
	}
	if err != nil {
		slog.Error("StartAuthzAgent failed. Continuing anyways", "err", err.Error())
		err = nil
	}
	return &agent, err
}
