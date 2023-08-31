package authservice

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"golang.org/x/exp/slog"
)

// AuthRolesCapability manages roles
// This implements the IAuthManageRoles interface.
//
// This implements the IAuthManageProfile interface.
type AuthRolesCapability struct {
	// Client record persistence
	store auth.IAuthnStore
	// message server for apply role changes
	msgServer msgserver.IMsgServer
}

// CreateRole adds a new custom role
func (svc *AuthRolesCapability) CreateRole(role string) error {
	// FIXME:implement
	slog.Error("CreateRole is not yet implemented")
	return nil
}

// DeleteRole deletes a custom role
func (svc *AuthRolesCapability) DeleteRole(role string) error {
	// FIXME:implement
	slog.Error("DeleteRole is not yet implemented")
	return nil
}

// SetRole sets a role for a client
func (svc *AuthRolesCapability) SetRole(clientID string, role string) error {
	// FIXME:validate role
	prof, err := svc.store.GetProfile(clientID)
	if err != nil {
		return err
	}
	prof.Role = role
	err = svc.store.Update(clientID, prof)
	return err
}

// NewAuthRolesCapability creates the auth role management capability
func NewAuthRolesCapability(store auth.IAuthnStore, msgServer msgserver.IMsgServer) *AuthRolesCapability {
	svc := AuthRolesCapability{store: store, msgServer: msgServer}
	return &svc
}
