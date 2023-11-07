package authservice

import (
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// AuthManageRoles manages custom roles.
// Intended for administrators.
//
// This implements the IAuthManageRoles interface.
type AuthManageRoles struct {
	// Client record persistence
	store authapi.IAuthnStore
	// message server for apply role changes
	msgServer msgserver.IMsgServer
	// action subscription
	actionSub transports.ISubscription
	// message server connection
	hc *hubclient.HubClient
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
func (svc *AuthManageRoles) HandleRequest(msg *things.ThingValue) (reply []byte, err error) {

	slog.Info("HandleRequest",
		slog.String("actionID", msg.Name),
		slog.String("senderID", msg.SenderID))
	switch msg.Name {
	case authapi.CreateRoleReq:
		req := &authapi.CreateRoleArgs{}
		err := ser.Unmarshal(msg.Data, &req)
		if err != nil {
			return nil, err
		}
		err = svc.CreateRole(req.Role)
		return nil, err
	case authapi.DeleteRoleReq:
		req := &authapi.DeleteRoleArgs{}
		err := ser.Unmarshal(msg.Data, &req)
		if err != nil {
			return nil, err
		}
		err = svc.DeleteRole(req.Role)
		return nil, err

	default:
		return nil, fmt.Errorf("unknown action '%s' for client '%s'", msg.Name, msg.SenderID)
	}
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (svc *AuthManageRoles) Start() (err error) {
	if svc.hc != nil {
		svc.actionSub, _ = svc.hc.SubRPCRequest(authapi.AuthRolesCapability, svc.HandleRequest)
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
	store authapi.IAuthnStore,
	hc *hubclient.HubClient,
	msgServer msgserver.IMsgServer) *AuthManageRoles {

	svc := AuthManageRoles{
		store:     store,
		hc:        hc,
		msgServer: msgServer,
	}
	return &svc
}
