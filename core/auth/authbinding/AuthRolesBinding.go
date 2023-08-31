package authbinding

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthRolesBinding binds the roles management service to a message based API
// This unmarshal requests and marshals responses
type AuthRolesBinding struct {
	svc   auth.IAuthManageRoles
	clSub hubclient.ISubscription
	hc    hubclient.IHubClient
}

// handle action requests directed at this capability
func (binding *AuthRolesBinding) handleActions(action *hubclient.ActionMessage) error {

	slog.Info("handleActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case auth.CreateRoleAction:
		req := &auth.CreateRoleReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.CreateRole(req.Role)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.DeleteRoleAction:
		req := &auth.DeleteRoleReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.DeleteRole(req.Role)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.SetRoleAction:
		req := &auth.SetRoleReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.SetRole(req.ClientID, req.Role)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("unknown action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (binding *AuthRolesBinding) Start() (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.clSub, _ = binding.hc.SubServiceCapability(auth.AuthRolesCapability, binding.handleActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthRolesBinding) Stop() {
	binding.clSub.Unsubscribe()
}

// NewAuthRolesBinding create a messaging binding for the role management service
//
//	svc is the authn svc to bind to.
//	hc is the server connection using the auth service credentials
func NewAuthRolesBinding(svc auth.IAuthManageRoles, hc hubclient.IHubClient) *AuthRolesBinding {
	an := &AuthRolesBinding{
		svc: svc,
		hc:  hc,
	}
	return an
}
