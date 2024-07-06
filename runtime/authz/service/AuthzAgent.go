package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// AuthzAgent serves the message based interface to the authz service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthzAgent struct {
	hc           hubclient.IHubClient
	svc          *AuthzService
	adminHandler hubclient.MessageHandler
	userHandler  hubclient.MessageHandler
}

// HandleMessage an event or action message for the authz service
// This message is send by the protocol client connected to this agent
func (agent *AuthzAgent) HandleMessage(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	var err error
	// if the message has an authn agent prefix then remove it.
	// This can happen if invoked directly through an embedded client
	_, thingID := things.SplitDigiTwinThingID(msg.ThingID)
	if thingID == authz.AdminServiceID {
		return agent.adminHandler(msg)
	} else if thingID == authz.UserServiceID {
		return agent.userHandler(msg)
	}
	err = fmt.Errorf("unknown authz service capability '%s'", msg.ThingID)
	stat.DeliveryFailed(msg, err)
	return stat
}

// StartAuthzAgent creates a new instance of the agent handling authorization service requests
// If hc is nil then use the HandleMessage method directly to pass messages to the agent,
// for example when testing.
//
//	svc is the authorization service whose capabilities to expose
//	hc is the optional message client used to publish and subscribe
func StartAuthzAgent(svc *AuthzService, hc hubclient.IHubClient) (*AuthzAgent, error) {
	var err error
	agent := AuthzAgent{svc: svc, hc: hc}
	agent.adminHandler = authz.NewAdminHandler(svc)
	agent.userHandler = authz.NewUserHandler(svc)

	if hc != nil {
		agent.hc.SetMessageHandler(agent.HandleMessage)
		// agents don't need to subscribe to receive actions directed at them
		//err = agent.hc.Subscribe(api.AuthzThingID)
	}

	// set permissions for using the authn services as authz wasn't yet running
	err = svc.SetPermissions(authn.AdminAgentID, authz.ThingPermissions{
		AgentID: authn.AdminAgentID,
		ThingID: authn.AdminServiceID,
		Allow:   []string{authn.ClientRoleService, authn.ClientRoleAdmin, authn.ClientRoleManager},
	})
	if err == nil {
		// all users with a role can GetProfile and refresh their token
		err = svc.SetPermissions(authn.UserAgentID, authz.ThingPermissions{
			AgentID: authn.UserAgentID,
			ThingID: authn.UserServiceID,
			Deny:    []string{authn.ClientRoleNone, ""},
		})
	}

	// set permissions for using the authz services
	if err == nil {
		err = svc.SetPermissions(authz.AdminAgentID, authz.ThingPermissions{
			AgentID: authz.AdminAgentID,
			ThingID: authz.AdminServiceID,
			Allow:   []string{authn.ClientRoleService, authn.ClientRoleAdmin, authn.ClientRoleManager},
		})
	}
	if err == nil {
		err = svc.SetPermissions(authz.UserAgentID, authz.ThingPermissions{
			AgentID: authz.UserAgentID,
			ThingID: authz.UserServiceID,
			Allow: []string{authn.ClientRoleAgent, authn.ClientRoleService,
				authn.ClientRoleAdmin, authn.ClientRoleManager, authn.ClientRoleOperator},
		})
	}
	if err != nil {
		slog.Error("StartAuthzAgent failed. Continuing anyways", "err", err.Error())
		err = nil
	}
	return &agent, err
}
