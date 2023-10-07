package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// AuthManageRoles manages custom roles.
// Intended for administrators.
//
// This implements the IAuthManageRoles interface.
type AuthManageRoles struct {
	// Client record persistence
	store auth.IAuthnStore
	// message server for apply role changes
	msgServer msgserver.IMsgServer
	// action subscription
	actionSub hubclient.ISubscription
	// message server connection
	hc hubclient.IHubClient
}

// CreateRole adds a new custom role
func (svc *AuthManageRoles) CreateRole(role string) error {
	// FIXME:implement
	slog.Error("CreateRole is not yet implemented")
	return nil
}

// DeleteRole deletes a custom role
func (svc *AuthManageRoles) DeleteRole(role string) error {
	// FIXME:implement
	slog.Error("DeleteRole is not yet implemented")
	return nil
}

// HandleRequest unmarshal and apply action requests
func (svc *AuthManageRoles) HandleRequest(action *hubclient.RequestMessage) error {

	slog.Info("handleActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case auth.CreateRoleReq:
		req := &auth.CreateRoleArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.CreateRole(req.Role)
		if err == nil {
			_ = action.SendAck()
		}
		return err
	case auth.DeleteRoleReq:
		req := &auth.DeleteRoleArgs{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = svc.DeleteRole(req.Role)
		if err == nil {
			_ = action.SendAck()
		}
		return err

	default:
		return fmt.Errorf("unknown action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (svc *AuthManageRoles) Start() (err error) {
	if svc.hc != nil {
		svc.actionSub, _ = svc.hc.SubServiceRPC(auth.AuthRolesCapability, svc.HandleRequest)
	}
	return err
}

// Stop removes subscriptions
func (svc *AuthManageRoles) Stop() {
	if svc.actionSub != nil {
		svc.actionSub.Unsubscribe()
		svc.actionSub = nil
	}
}

// NewAuthManageRoles creates the auth role management capability
func NewAuthManageRoles(
	store auth.IAuthnStore,
	hc hubclient.IHubClient,
	msgServer msgserver.IMsgServer) *AuthManageRoles {

	svc := AuthManageRoles{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
	}
	return &svc
}
