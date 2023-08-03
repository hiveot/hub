package authz

import "fmt"

// AuthzService handles client management and authorization for access to Things.
// This implements the IAuthz interface
//
// Authorization uses access control lists with group membership and roles to determine if a client
// is authorized to receive or post a message. This applies to all users of the message bus,
// regardless of how they are authenticated.
type AuthzService struct {
	aclStore  *AclFileStore
	authzAppl IAuthz
}

// GetPermissions returns a list of permissions a client has for a Thing
//func (authzService *AuthzService) GetPermissions(thingID string) (permissions []string, err error) {
//
//	return authzService.GetPermissions(clientID, thingID)
//}

// AddGroup adds a new group and creates a stream for it.
//
// publish to the connected stream.
func (svc *AuthzService) AddGroup(groupName string, retention uint64) error {
	err := svc.aclStore.AddGroup(groupName, retention)
	if err == nil {
		err = svc.authzAppl.AddGroup(groupName, retention)
	}
	return err
}

// AddService adds a client with the service role to a group
func (svc *AuthzService) AddService(serviceID string, groupName string) error {

	err := svc.aclStore.AddService(serviceID, groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.AddService(serviceID, groupName)
	}
	return err
}

// AddThing adds a client with the thing role to a group
func (svc *AuthzService) AddThing(thingID string, groupName string) error {

	err := svc.aclStore.AddThing(thingID, groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.AddThing(thingID, groupName)
	}
	return err
}

// AddUser adds a client with the user role to a group
func (svc *AuthzService) AddUser(userID string, role string, groupName string) (err error) {
	err = svc.aclStore.AddUser(userID, role, groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.AddUser(userID, role, groupName)
	}
	return err
}

// DeleteGroup deletes the group and associated resources. Use with care
func (svc *AuthzService) DeleteGroup(groupName string) error {
	err := svc.aclStore.DeleteGroup(groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.DeleteGroup(groupName)
	}
	return err
}

// GetClientRoles returns a map of [group]role for a client
func (svc *AuthzService) GetClientRoles(clientID string) (roles RoleMap, err error) {
	return svc.aclStore.GetClientRoles(clientID)
}

// GetGroup returns the group with the given name, or an error if group is not found.
// GroupName must not be empty
func (svc *AuthzService) GetGroup(groupName string) (group Group, err error) {

	group, err = svc.aclStore.GetGroup(groupName)
	return group, err
}

// GetPermissions returns a list of permissions a client has for Things
func (svc *AuthzService) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
	permissions = make(map[string][]string)
	for _, thingID := range thingIDs {
		var thingPerm []string
		clientRole, _ := svc.aclStore.GetRole(clientID, thingID)
		switch clientRole {
		case ClientRoleIotDevice:
		case ClientRoleThing:
			thingPerm = []string{PermPubEvents, PermReadActions}
			break
		case ClientRoleService:
			thingPerm = []string{PermPubActions, PermPubEvents, PermReadActions, PermReadEvents}
			break
		case ClientRoleManager:
			// managers are operators but can also change configuration
			// TODO: is publishing configuration changes a separate permission?
			thingPerm = []string{PermPubActions, PermReadEvents}
			break
		case ClientRoleOperator:
			thingPerm = []string{PermPubActions, PermReadEvents}
			break
		case ClientRoleViewer:
			thingPerm = []string{PermReadEvents}
			break
		default:
			thingPerm = []string{}
		}
		permissions[thingID] = thingPerm
	}
	return permissions, nil
}

// GetRole returns the highest role of a user has in groups shared with the thingID
// Intended to get client permissions in case of overlapping groups
func (svc *AuthzService) GetRole(clientID string, thingID string) (string, error) {
	return svc.aclStore.GetRole(clientID, thingID)
}

// ListGroups returns the list of known groups available to the client
func (svc *AuthzService) ListGroups(clientID string) (groups []Group, err error) {
	groups, err = svc.aclStore.ListGroups(clientID)
	return groups, err
}

// RemoveClient from a group
func (svc *AuthzService) RemoveClient(clientID string, groupName string) error {
	err := svc.aclStore.RemoveClient(clientID, groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.RemoveClient(clientID, groupName)
	}
	return err
}

// RemoveClientAll from all groups
func (svc *AuthzService) RemoveClientAll(clientID string) error {
	err := svc.aclStore.RemoveClientAll(clientID)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.RemoveClientAll(clientID)
	}
	return err
}

// SetUserRole sets the role for the user in a group
func (svc *AuthzService) SetUserRole(userID string, role string, groupName string) (err error) {
	err = svc.aclStore.SetUserRole(userID, role, groupName)
	if err == nil && svc.authzAppl != nil {
		err = svc.authzAppl.SetUserRole(userID, role, groupName)
	}
	return err
}

// Start returns an error if the store is missing
func (svc *AuthzService) Start() (err error) {
	if svc.aclStore == nil {
		return fmt.Errorf("Start: missing acl store")
	}
	return nil
}
func (svc *AuthzService) Stop() {
}

// NewAuthzService creates a new instance of the authorization service.
// Both store and application are provided to support dependency injection for other messaging systems.
//
//	aclStore persists the authorization rules
//	authzAppl applies the authorization configuration to the underlying messaging system. Use nil to ignore (for testing)
func NewAuthzService(aclStore *AclFileStore, authzAppl IAuthz) *AuthzService {

	authzService := AuthzService{
		aclStore:  aclStore,
		authzAppl: authzAppl,
	}
	return &authzService
}
