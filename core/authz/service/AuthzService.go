package service

import (
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/authz/service/aclstore"
	"golang.org/x/exp/slog"
	"strings"
)

// AuthzService handles client management and authorization for access to Things.
// This implements the IAuthz interface
//
// Authorization uses access control lists with group membership and roles to determine if a client
// is authorized to receive or post a message. This applies to all users of the message bus,
// regardless of how they are authenticated.
type AuthzService struct {
	aclStore *aclstore.AclFileStore
}

// GetPermissions returns a list of permissions a client has for a Thing
//func (authzService *AuthzService) GetPermissions(thingID string) (permissions []string, err error) {
//
//	return authzService.GetPermissions(clientID, thingID)
//}

// AddThing adds a Thing to a group
func (authzService *AuthzService) AddThing(thingID string, groupName string) error {

	err := authzService.aclStore.SetRole(thingID, groupName, authz.ClientRoleThing)
	return err
}

// GetPermissions returns a list of permissions a client has for a Thing
func (authzService *AuthzService) GetPermissions(clientID string, thingAddr string) (permissions []string, err error) {

	clientRole := authzService.aclStore.GetRole(clientID, thingAddr)
	switch clientRole {
	case authz.ClientRoleIotDevice:
		permissions = []string{authz.PermReadAction, authz.PermPubEvent, authz.PermPubTD}
	case authz.ClientRoleManager:
		permissions = []string{authz.PermEmitAction, authz.PermReadEvent, authz.PermReadAction,
			authz.PermReadTD, authz.PermWriteProperty}
	case authz.ClientRoleOperator:
		permissions = []string{authz.PermEmitAction, authz.PermReadEvent, authz.PermReadAction, authz.PermReadTD}
	case authz.ClientRoleViewer:
		permissions = []string{authz.PermReadEvent, authz.PermReadAction, authz.PermReadTD}
	default:
		permissions = []string{}
	}
	return permissions, nil
}

// IsPublisher checks if the deviceID is the publisher of the thingAddr.
// This requires that the thingAddr is formatted as publisherID/thingID
// Returns true if the deviceID is the publisher of the thingID, false if not.
func (authzService *AuthzService) IsPublisher(deviceID string, thingAddr string) (bool, error) {

	// FIXME use a helper for this so the domain knownledge is concentraged
	addrParts := strings.Split(thingAddr, "/")
	return addrParts[0] == deviceID, nil
}

// GetGroup returns the group with the given name, or an error if group is not found.
// GroupName must not be empty
func (authzService *AuthzService) GetGroup(groupName string) (group authz.Group, err error) {

	group, err = authzService.aclStore.GetGroup(groupName)
	return group, err
}

// GetGroupRoles returns a list of roles in groups the client is a member of.
func (authzService *AuthzService) GetGroupRoles(clientID string) (roles authz.RoleMap, err error) {

	// simple pass through
	roles = authzService.aclStore.GetGroupRoles(clientID)
	return roles, nil
}

// ListGroups returns the list of known groups
func (authzService *AuthzService) ListGroups(limit int, offset int) (groups []authz.Group, err error) {

	groups = authzService.aclStore.ListGroups(limit, offset)
	return groups, nil
}

// RemoveAll from all groups
func (authzService *AuthzService) RemoveAll(clientID string) error {
	err := authzService.aclStore.RemoveAll(clientID)
	return err
}

// RemoveClient from a group
func (authzService *AuthzService) RemoveClient(clientID string, groupName string) error {
	err := authzService.aclStore.Remove(clientID, groupName)
	return err
}

// RemoveThing removes a Thing from a group
func (authzService *AuthzService) RemoveThing(thingID string, groupName string) error {

	err := authzService.aclStore.Remove(thingID, groupName)
	return err
}

// SetClientRole sets the role for the client in a group
func (authzService *AuthzService) SetClientRole(clientID string, groupName string, role string) error {
	err := authzService.aclStore.SetRole(clientID, groupName, role)
	return err
}

// Stop closes the service and release resources
func (authzService *AuthzService) Stop() {
	authzService.aclStore.Close()
}

// Start the ACL store for reading
func (authzService *AuthzService) Start() error {
	slog.Info("Opening ACL store")
	err := authzService.aclStore.Open()
	if err != nil {
		return err
	}
	return nil
}

// NewAuthzService creates a new instance of the authorization service.
//
//	aclStore provides the functions to read and write authorization rules
func NewAuthzService(aclStorePath string) *AuthzService {
	aclStore := aclstore.NewAclFileStore(aclStorePath, authz.ServiceName)

	authzService := AuthzService{
		aclStore: aclStore,
	}
	return &authzService
}
