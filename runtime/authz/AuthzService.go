package authz

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
)

// AuthzService is the authorization service for authorizing access to devices
type AuthzService struct {

	// configuration with role
	cfg *AuthzConfig

	// authz currently uses authn store to persist the user's role
	// this is good enough as long as a user only has a single role
	authnStore api.IAuthnStore
}

// CreateRole adds a new custom role
//func (svc *AuthzService) CreateRole(role string) error {
//	// FIXME:implement
//	slog.Error("CreateRole is not yet implemented")
//	return nil
//}

// DeleteRole deletes a custom role
//func (svc *AuthzService) DeleteRole(role string) error {
//	// FIXME:implement
//	slog.Error("DeleteRole is not yet implemented")
//	return nil
//}

// CanPubAction checks if the given client can publish actions
func (svc *AuthzService) CanPubAction(clientID string) bool {
	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeAction, true)
	return hasPerm
}

// CanPubEvent checks if the given client can publish events
func (svc *AuthzService) CanPubEvent(clientID string) bool {
	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeEvent, true)
	return hasPerm
}

// CanSubAction checks if the given client can subscribe to actions
func (svc *AuthzService) CanSubAction(clientID string) bool {
	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeAction, false)
	return hasPerm
}

// CanSubEvent checks if the given client can subscribe to events
func (svc *AuthzService) CanSubEvent(clientID string) bool {
	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeEvent, false)
	return hasPerm
}

// GetClientRole returns the role assigned to the client or an error
func (svc *AuthzService) GetClientRole(clientID string) (string, error) {
	// this simply returns the default role stored with the client
	// in future more roles could be added in which case authz will need its own store.
	role, err := svc.authnStore.GetRole(clientID)
	return role, err
}

// GetRolePermissions returns the permissions for the given role
//func (svc *AuthzService) GetRolePermissions(role string) ([]RolePermission, bool) {
//	rolePerm, found := svc.cfg.rolePermissions[role]
//	return rolePerm, found
//}

// HasPermission returns whether the client has permission to pub or sub a message type
//
//	clientID must be an authenticated client and have a role assigned
//	messageType to check: MessageTypeAction/Event/RPC
//	isPub true to check for publish permissions, false for subscribe permission
//
// This returns true if the client has permission, false if the client does not have the permission
func (svc *AuthzService) HasPermission(clientID string, messageType string, isPub bool) bool {
	role, err := svc.GetClientRole(clientID)
	if err != nil || role == "" {
		// unknown client or missing role
		return false
	}
	rolePerms, found := svc.cfg.rolePermissions[role]
	if !found {
		return false
	}
	// pick the first match. This doesn't check for agent, thing/interface, or key/method
	for _, perm := range rolePerms {
		if isPub && perm.AllowPub &&
			messageType == perm.MsgType {
			return true
		} else if !isPub && perm.AllowSub && messageType == perm.MsgType {
			return true
		}
	}
	return false
}

// SetClientRole sets the role of a client in the authz store
func (svc *AuthzService) SetClientRole(clientID string, role string) error {
	return svc.authnStore.SetRole(clientID, role)
}

// Start starts the authorization service
func (svc *AuthzService) Start() error {
	slog.Info("Starting AuthzService")
	return nil
}

// Stop stops the authorization service
func (svc *AuthzService) Stop() {
	slog.Info("Stopping AuthzService")
}

// NewAuthzService creates a new instance of the authorization service with default rules
//
//	authnStore is used to store default client roles
func NewAuthzService(cfg *AuthzConfig, authnStore api.IAuthnStore) *AuthzService {
	svc := &AuthzService{
		cfg:        cfg,
		authnStore: authnStore,
	}
	return svc
}

// StartAuthzService creates and start the authz administration service
// with the given config.
// This uses the authn store to store the user role
func StartAuthzService(cfg *AuthzConfig, authnStore api.IAuthnStore) (*AuthzService, error) {

	svc := NewAuthzService(cfg, authnStore)
	err := svc.Start()
	return svc, err
}
