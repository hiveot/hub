package authzservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slog"
	"path"
	"time"
)

// AuthzService handles client management and authorization for access to Things.
// This implements the IAuthz interface
//
// Authorization uses access control lists with group membership and roles to determine if a client
// is authorized to receive or post a message. This applies to all users of the message bus,
// regardless of how they are authenticated.
type AuthzService struct {
	aclStore     *AclFileStore
	msgServer    msgserver.IMsgServer
	hc           hubclient.IHubClient
	authzBinding AuthzBinding
}

// GetPermissions returns a list of permissions a client has for a Thing
//func (authzService *AuthzService) GetPermissions(thingID string) (permissions []string, err error) {
//
//	return authzService.GetPermissions(clientID, thingID)
//}

// AddGroup adds a new group and creates a stream for it.
// if groupName is the all group then it events from all things are added
// publish to the connected stream.
// if retention is less than 1 second, it is set to  DefaultGroupRetention
func (svc *AuthzService) AddGroup(groupID string, DisplayName string, retention time.Duration) error {
	if retention <= time.Second {
		retention = time.Duration(authz.DefaultGroupRetention * time.Second)
	}
	err := svc.aclStore.AddGroup(groupID, DisplayName, retention)
	svc.onGroupsChange()
	return err
}

// AddService adds a client with the service role to a group
func (svc *AuthzService) AddService(serviceID string, groupID string) error {

	err := svc.aclStore.AddService(serviceID, groupID)
	svc.onChange()
	return err
}

// AddThing adds a client with the thing role to a group
func (svc *AuthzService) AddThing(thingID string, groupID string) error {

	err := svc.aclStore.AddThing(thingID, groupID)
	svc.onChange()
	// FIXME: this doesn't belong here.
	// in NATS things are added as a group source so the group needs to be reloaded
	svc.onGroupsChange()
	return err
}

// AddDevice adds all things from a device to a group
func (svc *AuthzService) AddDevice(deviceID string, groupID string) error {

	err := svc.aclStore.AddDevice(deviceID, groupID)
	svc.onChange()
	// FIXME: this doesn't belong here.
	// in NATS things are added as a group source so the group needs to be reloaded
	svc.onGroupsChange()
	return err
}

// AddUser adds a client with the user role to a group
func (svc *AuthzService) AddUser(userID string, role string, groupID string) (err error) {
	err = svc.aclStore.AddUser(userID, role, groupID)
	svc.onChange()
	return err
}

// DeleteGroup deletes the group and associated resources. Use with care
func (svc *AuthzService) DeleteGroup(groupID string) error {
	err := svc.aclStore.DeleteGroup(groupID)
	svc.onGroupsChange()
	return err
}

// GetClientGroups returns the list of known groups available to the client
func (svc *AuthzService) GetClientGroups(clientID string) (groups []authz.Group, err error) {
	groups, err = svc.aclStore.ListGroups(clientID)
	return groups, err
}

// GetClientRoles returns a map of [group]role for a client
func (svc *AuthzService) GetClientRoles(clientID string) (roles authz.RoleMap, err error) {
	return svc.aclStore.GetClientRoles(clientID)
}

// GetGroup returns the group with the given name, or an error if group is not found.
// GroupName must not be empty
func (svc *AuthzService) GetGroup(groupID string) (group authz.Group, err error) {

	group, err = svc.aclStore.GetGroup(groupID)
	return group, err
}

// GetPermissions returns a list of permissions a client has for Things
func (svc *AuthzService) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
	permissions = make(map[string][]string)
	for _, thingID := range thingIDs {
		var thingPerm []string
		clientRole, _ := svc.aclStore.GetRole(clientID, thingID)
		switch clientRole {
		case authz.GroupRoleIotDevice:
		case authz.GroupRoleThing:
			thingPerm = []string{authz.PermPubEvents, authz.PermReadActions}
			break
		case authz.GroupRoleService:
			thingPerm = []string{authz.PermPubActions, authz.PermPubEvents, authz.PermReadActions, authz.PermReadEvents}
			break
		case authz.GroupRoleManager:
			// managers are operators but can also change configuration
			// TODO: is publishing configuration changes a separate permission?
			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
			break
		case authz.GroupRoleOperator:
			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
			break
		case authz.GroupRoleViewer:
			thingPerm = []string{authz.PermReadEvents}
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

// notification handler invoked when groups have changed
func (svc *AuthzService) onGroupsChange() {
	groupList := make([]authz.Group, 0, len(svc.aclStore.groups))
	for _, grp := range svc.aclStore.groups {
		groupList = append(groupList, grp)
	}
	_ = svc.msgServer.ApplyGroups(groupList)
}

// notification handler invoked when client permissions have changed
// this invokes a reload of server authz
func (svc *AuthzService) onChange() {
	_ = svc.msgServer.ApplyAuthz(svc.aclStore.clientGroupRoles)
}

// RemoveClient from a group
func (svc *AuthzService) RemoveClient(clientID string, groupName string) error {
	err := svc.aclStore.RemoveClient(clientID, groupName)
	svc.onChange()
	return err
}

// RemoveClientAll from all groups
func (svc *AuthzService) RemoveClientAll(clientID string) error {
	err := svc.aclStore.RemoveClientAll(clientID)
	svc.onChange()
	return err
}

// SetUserRole sets the role for the user in a group
func (svc *AuthzService) SetUserRole(userID string, role string, groupName string) (err error) {
	err = svc.aclStore.SetUserRole(userID, role, groupName)
	svc.onChange()
	return err
}

// Start the service. This:
// 1. opens the acl store for group persistence
// 2. starts the server adapter for applying changes to the server
// 3. creates the 'all' group if it doesn't yet exist
// 4. starts the messaging binding to listen for action requests
func (svc *AuthzService) Start() (err error) {

	if svc.msgServer == nil || svc.aclStore == nil {
		return fmt.Errorf("missing acl store or message server")
	}
	svc.hc, err = svc.msgServer.ConnectInProc("authz")
	if err != nil {
		return fmt.Errorf("can't connect authz to server: %w", err)
	}

	// the binding subscribes to the message bus to receive action requests,
	// (un)marshals action messages and invokes the authz service with requests.
	svc.authzBinding = NewAuthzBinding(svc, svc.hc)

	if svc.aclStore == nil {
		return fmt.Errorf("start: missing acl store")
	}
	err = svc.aclStore.Open()
	if err != nil {
		return err
	}
	err = svc.authzBinding.Start()
	if err != nil {
		return err
	}
	// ensure that the all group exists and all devices are a member
	_, err = svc.GetGroup(authz.AllGroupID)
	if err != nil {
		// add all things to the group. this can be done before adding the group itself
		err = svc.AddThing("", authz.AllGroupID)
		if err != nil {
			slog.Error("failed adding things to all group", "err", err.Error())
		}
		// TBD: best retention for the all group
		// the all group subscribes to all events
		err = svc.AddGroup(authz.AllGroupID, "All Things Group", time.Hour*24*31)
	}

	return err
}

func (svc *AuthzService) Stop() {
	svc.authzBinding.Stop()
	if svc.aclStore != nil {
		svc.aclStore.Close()
	}
	if svc.hc != nil {
		svc.hc.Disconnect()
	}
}

// NewAuthzService creates a new instance of the authorization service.
// The provided store and adapter instances will be started on Start() and ended on Stop()
//
//	aclStore persists the authorization rules
//	hc is the server connection used to subscribe
func NewAuthzService(aclStore *AclFileStore, msgServer msgserver.IMsgServer) *AuthzService {

	authzService := &AuthzService{
		aclStore:  aclStore,
		msgServer: msgServer,
	}

	return authzService
}

// StartAuthzService creates and launch the authz service with the given config
// This creates a password store using the config file and password encryption method.
func StartAuthzService(cfg AuthzConfig, msgServer msgserver.IMsgServer) (*AuthzService, error) {
	aclFile := path.Join(cfg.DataDir, authz.DefaultAclFilename)
	aclStore := NewAuthzFileStore(aclFile)
	authzSvc := NewAuthzService(aclStore, msgServer)
	err := authzSvc.Start()
	if err != nil {
		logrus.Panicf("cant start test authn service: %s", err)
	}
	return authzSvc, err
}
