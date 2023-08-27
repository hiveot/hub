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

// notification handler invoked when groups have changed
func (svc *AuthzService) _onGroupsChange() {
	groupList := make([]authz.Group, 0, len(svc.aclStore.groups))
	for _, grp := range svc.aclStore.groups {
		groupList = append(groupList, grp)
	}
	_ = svc.msgServer.ApplyGroups(groupList)
}

// notification handler invoked when client permissions have changed
// this invokes a reload of server authz
func (svc *AuthzService) _onChange() {
	_ = svc.msgServer.ApplyAuthz(svc.aclStore.userGroupRoles)
}

// AddSource adds an event source with the thing role to a group
//
//	publisherID is the device or service that publishes the events
//	thingID is the Thing whose info is published or "" for all things of the publisher
func (svc *AuthzService) AddSource(publisherID string, thingID string, groupID string) error {

	slog.Info("AddSource",
		slog.String("publisherID", publisherID),
		slog.String("thingID", thingID),
		slog.String("groupID", groupID))
	err := svc.aclStore.AddSource(publisherID, thingID, groupID)
	//svc._onChange()
	svc._onGroupsChange()
	return err
}

// AddUser adds a consumer to a group with the user role:
//
//	manager, operator viewer or service.
//
// If the client is already in the group this returns without error
// See ClientRole...
func (svc *AuthzService) AddUser(userID string, role string, groupID string) (err error) {
	slog.Info("AddUser",
		slog.String("userID", userID),
		slog.String("role", role),
		slog.String("groupID", groupID))
	err = svc.aclStore.AddUser(userID, role, groupID)
	svc._onChange()
	return err
}

// CreateGroup adds a new group
// If the group exists this returns without error
// Use retention 0 to retain messages indefinitely
//
//	groupID unique ID of the group
//	displayName of the group
//	retention period of events in this group. 0 for DefaultGroupRetention
func (svc *AuthzService) CreateGroup(groupID string, displayName string, retention time.Duration) error {
	slog.Info("CreateGroup",
		slog.String("displayName", displayName),
		slog.String("groupID", groupID),
		slog.Duration("retention", retention),
	)
	if retention <= time.Second {
		retention = time.Duration(authz.DefaultGroupRetention * time.Second)
	}
	err := svc.aclStore.AddGroup(groupID, displayName, retention)
	svc._onGroupsChange()
	return err
}

// DeleteGroup deletes the group and associated resources. Use with care
func (svc *AuthzService) DeleteGroup(groupID string) error {
	slog.Info("DeleteGroup", slog.String("groupID", groupID))
	err := svc.aclStore.DeleteGroup(groupID)
	svc._onGroupsChange()
	return err
}

// GetGroup returns the group with the given name, or an error if group is not found.
// groupID must not be empty and must be an existing group
// Returns an error if the group does not exist.
func (svc *AuthzService) GetGroup(groupID string) (group authz.Group, err error) {
	group, err = svc.aclStore.GetGroup(groupID)
	return group, err
}

// GetUserGroups returns the list of groups the user is a member of
// If userID is "" then all groups are returned.
func (svc *AuthzService) GetUserGroups(clientID string) (groups []authz.Group, err error) {
	groups, err = svc.aclStore.GetUserGroups(clientID)
	return groups, err
}

// GetUserRoles returns a map of [group]role for a client
func (svc *AuthzService) GetUserRoles(userID string) (roles authz.UserRoleMap, err error) {
	return svc.aclStore.GetUserRoles(userID)
}

// GetPermissions returns a list of permissions a client has for Things
//func (svc *AuthzService) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
//	permissions = make(map[string][]string)
//	for _, thingID := range thingIDs {
//		var thingPerm []string
//		clientRole, _ := svc.aclStore.GetRole(clientID, thingID)
//		switch clientRole {
//		case authz.GroupRoleIotDevice:
//		case authz.GroupRoleThing:
//			thingPerm = []string{authz.PermPubEvents, authz.PermReadActions}
//			break
//		case authz.UserRoleService:
//			thingPerm = []string{authz.PermPubActions, authz.PermPubEvents, authz.PermReadActions, authz.PermReadEvents}
//			break
//		case authz.UserRoleManager:
//			// managers are operators but can also change configuration
//			// TODO: is publishing configuration changes a separate permission?
//			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
//			break
//		case authz.UserRoleOperator:
//			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
//			break
//		case authz.UserRoleViewer:
//			thingPerm = []string{authz.PermReadEvents}
//			break
//		default:
//			thingPerm = []string{}
//		}
//		permissions[thingID] = thingPerm
//	}
//	return permissions, nil
//}

//// GetRole returns the highest role of a user has in groups shared with the thingID
//// Intended to get client permissions in case of overlapping groups
//func (svc *AuthzService) GetRole(clientID string, thingID string) (string, error) {
//	return svc.aclStore.GetRole(clientID, thingID)
//}

// RemoveSource removes a source from a group
// If the source doesn't exist then returns without error
// The caller must be an administrator or service.
//
//	publisherID is required and identifies the publisher of the event
//	thingID is optional and identifies a specific Thing, "" for all things of the publisher
func (svc *AuthzService) RemoveSource(publisherID, thingID string, groupID string) error {
	slog.Info("RemoveSource",
		slog.String("publisherID", publisherID),
		slog.String("thingID", thingID),
		slog.String("groupID", groupID))
	err := svc.aclStore.RemoveSource(publisherID, thingID, groupID)
	svc._onChange()
	return err
}

// RemoveUser removes a user from a group
// If the user doesn't exist then returns without error
// The caller must be an administrator or service.
func (svc *AuthzService) RemoveUser(userID string, groupID string) error {
	slog.Info("RemoveUser",
		slog.String("userID", userID),
		slog.String("groupID", groupID))
	err := svc.aclStore.RemoveUser(userID, groupID)
	svc._onChange()
	return err
}

// RemoveUserAll removes a user from all groups.
// If the user doesn't exist then returns without error
// The caller must be an administrator or service.
func (svc *AuthzService) RemoveUserAll(userID string) error {
	slog.Info("RemoveUserAll", slog.String("userID", userID))
	err := svc.aclStore.RemoveUserAll(userID)
	svc._onChange()
	return err
}

// SetUserRole sets the role for the user in a group
func (svc *AuthzService) SetUserRole(userID string, role string, groupID string) (err error) {
	slog.Info("SetUserRole",
		slog.String("userID", userID),
		slog.String("role", role),
		slog.String("groupID", groupID),
	)
	err = svc.aclStore.SetUserRole(userID, role, groupID)
	svc._onChange()
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
		// Add all devices and things to the group.
		// This can be done before adding the group itself
		err = svc.AddSource("", "", authz.AllGroupID)
		if err != nil {
			slog.Error("failed adding things to all group", "err", err.Error())
		}
		// TBD: best retention for the all group
		// the all group subscribes to all events
		err = svc.CreateGroup(authz.AllGroupID, "All Things Group", time.Hour*24*31)
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
