package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	authz2 "github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authz"
	"log/slog"
)

// AuthzService is the authorization service for authorizing access to devices
type AuthzService struct {

	// configuration with role
	cfg *authz.AuthzConfig

	// authz currently uses authn store to persist the user's role
	// this is good enough as long as a user only has a single role
	authnStore api.IAuthnStore
}

// CreateCustomRole adds a new custom role
func (svc *AuthzService) CreateCustomRole(role string) error {
	// FIXME:implement
	slog.Error("CreateRole is not yet implemented")
	return nil
}

// DeleteCustomRole deletes a custom role
func (svc *AuthzService) DeleteCustomRole(role string) error {
	// FIXME:implement
	slog.Error("DeleteRole is not yet implemented")
	return nil
}

//
//// CanPubAction checks if the given client can publish actions
//func (svc *AuthzService) CanPubAction(clientID string) bool {
//	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeAction, true)
//	return hasPerm
//}
//
//// CanPubEvent checks if the given client can publish events
//func (svc *AuthzService) CanPubEvent(clientID string) bool {
//	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeEvent, true)
//	return hasPerm
//}
//
//// CanSubAction checks if the given client can subscribe to actions
//func (svc *AuthzService) CanSubAction(clientID string) bool {
//	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeAction, false)
//	return hasPerm
//}
//
//// CanSubEvent checks if the given client can subscribe to events
//func (svc *AuthzService) CanSubEvent(clientID string) bool {
//	hasPerm := svc.HasPermission(clientID, vocab.MessageTypeEvent, false)
//	return hasPerm
//}

// GetClientRole returns the role assigned to the client or an error
func (svc *AuthzService) GetClientRole(senderID string, clientID string) (string, error) {
	// this simply returns the default role stored with the client
	// in future more roles could be added in which case authz will need its own store.
	role, err := svc.authnStore.GetRole(clientID)
	return role, err
}

// GetRolePermissions returns the permissions for the given role
func (svc *AuthzService) GetRolePermissions(senderID string, role string) ([]api.RolePermission, bool) {
	rolePerm, found := svc.cfg.RolePermissions[role]
	return rolePerm, found
}

// SetClientRole sets the role of a client in the authz store
func (svc *AuthzService) SetClientRole(senderID string, args authz2.AdminSetClientRoleArgs) error {
	// okay, we lied, it uses the authn store
	return svc.authnStore.SetRole(args.ClientID, args.Role)
}

// SetPermissions sets the client roles that are allowed to use an agent's service.
// Intended for use by services and agents to set the roles that have access to it.
// This fails if the caller is not a service or agent.
//
//	senderID is the agent or service that sets the permissions
//	perms are the permissions being set
func (svc *AuthzService) SetPermissions(senderID string, perms authz2.ThingPermissions) error {
	// the sender must be a service
	slog.Info("SetPermissions",
		slog.String("senderID", senderID),
		slog.String("agentID", perms.AgentID),
		slog.String("thingID", perms.ThingID))

	clientProfile, err := svc.authnStore.GetProfile(senderID)
	role, _ := svc.authnStore.GetRole(senderID)
	if err != nil {
		return err
	} else if role == authn.ClientRoleAdmin {
		// administrators can set permissions for others
		slog.Info("Administrator setting role")
	} else if senderID != perms.AgentID {
		// unless the sender is an admin (todo), it cannot set permissions for someone else
		return fmt.Errorf(
			"sender '%s' cannot set permissions for agent '%s'", senderID, perms.AgentID)
	} else if clientProfile.ClientType == authn.ClientTypeConsumer {
		return fmt.Errorf(
			"'%s' is a consumer and consumers cannot set permissions", senderID)
	}
	// store the permissions
	svc.cfg.SetPermissions(perms)
	return nil
}

// Start starts the authorization service
func (svc *AuthzService) Start() error {
	// fixme: set permissions
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
func NewAuthzService(cfg *authz.AuthzConfig, authnStore api.IAuthnStore) *AuthzService {
	svc := &AuthzService{
		cfg:        cfg,
		authnStore: authnStore,
	}
	return svc
}

// StartAuthzService creates and start the authz administration service
// with the given config.
// This uses the authn store to store the user role
func StartAuthzService(cfg *authz.AuthzConfig, authnStore api.IAuthnStore) (*AuthzService, error) {

	svc := NewAuthzService(cfg, authnStore)
	err := svc.Start()
	return svc, err
}
