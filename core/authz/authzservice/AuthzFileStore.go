package authzservice

import (
	"bufio"
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AclFileStore is an in-memory ACL store based on the state store
// This tracks client groups in two indexes, one by group and one by clientID
type AclFileStore struct {
	serviceID string

	// Groups is an index of ACL groups by their name. Stored
	// map[groupID]Group
	groups map[string]authz.Group `yaml:"groups"`

	// state store file
	storePath string

	// index of users and their group roles. Updated on load.
	// intended for fast lookup of roles
	userGroupRoles map[string]authz.UserRoleMap //[userID][groupID]role
	mutex          sync.RWMutex
}

// _buildUserGroupRoles builds a map of userIDs with user roles
// intended for fast lookup
func (aclStore *AclFileStore) _buildUserGroupRoles() map[string]authz.UserRoleMap {

	// build the role map index for each user in the groups: map[userID]map[groupID]role
	userGroupRoles := make(map[string]authz.UserRoleMap)
	// for each group, add its members to the user index
	for groupID, group := range aclStore.groups {
		// iterate the group members and add them to the client index along with its group role
		for memberID, memberRole := range group.MemberRoles {
			//
			groupRoles, found := userGroupRoles[memberID]
			if !found {
				// Need to add the set of group roles for this user
				groupRoles = make(authz.UserRoleMap)
				userGroupRoles[memberID] = groupRoles
			}
			groupRoles[groupID] = memberRole
		}
	}
	return userGroupRoles
}

// AddGroup adds a new group to the store
// retention is the time data is kept in the group, 0 for indefinitely
func (aclStore *AclFileStore) AddGroup(groupID string, displayName string, retention time.Duration) error {
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()
	_, exists := aclStore.groups[groupID]
	if exists {
		return nil
	}
	aclStore.groups[groupID] = authz.Group{
		ID:          groupID,
		DisplayName: displayName,
		MemberRoles: authz.UserRoleMap{},
		Sources:     make([]authz.EventSource, 0),
		Retention:   retention,
	}
	err := aclStore.Save()
	return err
}

// AddDevice adds all things from a publishing device or service as a group source
//func (aclStore *AclFileStore) AddDevice(deviceID string, groupID string) error {
//	err := aclStore.setRole(deviceID, authz.GroupRoleIotDevice, groupID)
//	return err
//}

// AddService adds a client with the service role to a group
//func (aclStore *AclFileStore) AddService(serviceID string, groupID string) error {
//	err := aclStore.setRole(serviceID, authz.UserRoleService, groupID)
//	return err
//}

// AddSource adds a thing as a group source
func (aclStore *AclFileStore) AddSource(
	publisherID string, thingID string, groupID string) error {

	// update the group
	group, found := aclStore.groups[groupID]
	if !found {
		group = authz.Group{
			ID:          groupID,
			Sources:     []authz.EventSource{},
			MemberRoles: authz.UserRoleMap{},
		}
		aclStore.groups[groupID] = group
	}
	group.Sources = append(group.Sources, authz.EventSource{
		PublisherID: publisherID,
		ThingID:     thingID,
	})
	aclStore.groups[groupID] = group

	// save
	err := aclStore.Save()

	//slog.Info("AclFileStore.AddSource",
	//	"publisherID", publisherID, "thingID", thingID, "group", groupID)
	return err
}

// AddUser adds a client with the user role to a group
func (aclStore *AclFileStore) AddUser(userID string, role string, groupID string) (err error) {

	if role == authz.UserRoleViewer ||
		role == authz.UserRoleManager ||
		role == authz.UserRoleOperator ||
		role == authz.UserRoleService {

		err = aclStore.setUserRole(userID, role, groupID)
	} else {
		err = fmt.Errorf("role '%s' doesn't apply to users", role)
	}
	return err
}

// Close the store
func (aclStore *AclFileStore) Close() {
	slog.Info("AclFileStore.Release", "serviceID", aclStore.serviceID)
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()
	//if aclStore.watcher != nil {
	//	_ = aclStore.watcher.Close()
	//	aclStore.watcher = nil
	//}
}

// DeleteGroup deletes the given group from the store
func (aclStore *AclFileStore) DeleteGroup(groupID string) error {
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	delete(aclStore.groups, groupID)
	aclStore.userGroupRoles = aclStore._buildUserGroupRoles()

	err := aclStore.Save()

	slog.Info("group removed", "groupID", groupID)
	return err
}

// GetGroup returns the group of the given name
func (aclStore *AclFileStore) GetGroup(groupID string) (authz.Group, error) {

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()
	group, found := aclStore.groups[groupID]
	if !found {
		err := fmt.Errorf("group '%s' does not exist", groupID)
		return group, err
	}
	return group, nil
}

//// GetHighestRole returns the highest role of a user has in a list of group
//// Intended to get client permissions in case of overlapping groups
//func (aclStore *AclFileStore) GetHighestRole(clientID string, groupIDs []string) string {
//	highestRole := authz.UserRoleNone
//
//	aclStore.mutex.RLock()
//	defer aclStore.mutex.RUnlock()
//
//	groupRoles := aclStore.userGroupRoles[clientID]
//
//	for _, groupID := range groupIDs {
//		clientRole, found := groupRoles[groupID]
//		if found && IsRoleGreaterEqual(clientRole, highestRole) {
//			highestRole = clientRole
//		}
//	}
//	return highestRole
//}

// GetUserGroups returns the list of groups the user is a member of
// If userID is "" then all groups are returned.
func (aclStore *AclFileStore) GetUserGroups(userID string) (groups []authz.Group, err error) {
	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	// get all groups
	if userID == "" {
		groups = make([]authz.Group, 0, len(aclStore.groups))
		for _, group := range aclStore.groups {
			groups = append(groups, group)
		}
		return groups, nil
	}
	groupRoles, found := aclStore.userGroupRoles[userID]
	if !found {
		return nil, fmt.Errorf("user '%s' not found", userID)
	}
	groups = make([]authz.Group, 0, len(groupRoles))
	for groupID, _ := range groupRoles {
		group, found := aclStore.groups[groupID]
		if !found {
			slog.Error("user is a member of group but the group no longer exists",
				"groupID", groupID, "userID", userID)
		} else {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

// GetUserRoles returns the roles a user has in various groups
func (aclStore *AclFileStore) GetUserRoles(clientID string) (authz.UserRoleMap, error) {

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	roles := aclStore.userGroupRoles[clientID]

	return roles, nil
}

// GetUsersGroupsRoles returns the roles users have in each group
// This returns a map of clients with their groups and role in that group
//func (aclStore *AclFileStore) GetUsersGroupsRoles() map[string]authz.RoleMap {
//	shallowCopy := make(map[string]authz.RoleMap)
//	aclStore.mutex.RLock()
//	defer aclStore.mutex.RUnlock()
//
//	aclStore.mutex.RLock()
//	for userID, roles := range aclStore.userGroupRoles {
//		shallowCopy[userID] = roles
//	}
//	aclStore.mutex.RUnlock()
//
//	return shallowCopy
//}

// GetPermissions returns a list of permissions a client has for Things
//func (aclStore *AclFileStore) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
//	permissions = make(map[string][]string)
//	for _, thingID := range thingIDs {
//		var thingPerm []string
//		clientRole, _ := aclStore.GetRole(clientID, thingID)
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

// GetRole returns the highest role of a user has in groups shared with the thingID
// Intended to get client permissions in case of overlapping groups
//func (aclStore *AclFileStore) GetRole(clientID string, thingID string) (string, error) {
//	groups := aclStore.GetSharedGroups(clientID, thingID)
//	highestRole := aclStore.GetHighestRole(clientID, groups)
//	return highestRole, nil
//}

// GetSharedGroups returns a list of groups both a client and thing are a member of.
//func (aclStore *AclFileStore) GetSharedGroups(clientID string, thingID string) []string {
//	sharedGroups := []string{}
//
//	aclStore.mutex.RLock()
//	defer aclStore.mutex.RUnlock()
//	// iterate the groups of the client
//	groupRoles, clientHasMemberships := aclStore.userGroupRoles[clientID]
//	if !clientHasMemberships {
//		return sharedGroups
//	}
//	for groupID := range groupRoles {
//		// client is a member of this group, check if the thingID is also a member
//		_, thingIsMember := aclStore.userGroupRoles[thingID]
//		// all things are a member of the all group
//		if thingIsMember || groupID == authz.AllGroupID {
//			sharedGroups = append(sharedGroups, groupID)
//		}
//	}
//	return sharedGroups
//}

// IsRoleGreaterEqual returns true if a user role has same or greater permissions
// than the minimum role.
func IsRoleGreaterEqual(role string, minRole string) bool {
	if minRole == authz.UserRoleNone || role == minRole {
		return true
	}
	if minRole == authz.UserRoleViewer && role != authz.UserRoleNone {
		return true
	}
	if minRole == authz.UserRoleOperator && (role == authz.UserRoleManager) {
		return true
	}
	return false
}

// ListGroups return ... a list of all groups available to the client
//
//	consumerID client that is a member of the groups to return, or "" for all groups
//func (aclStore *AclFileStore) ListGroups(consumerID string) ([]authz.Group, error) {
//	groups := make([]authz.Group, 0, len(aclStore.groups))
//	for _, group := range aclStore.groups {
//		_, found := group.MemberRoles[clientID]
//		if found || clientID == "" {
//			groups = append(groups, group)
//		}
//	}
//	return groups, nil
//}

// Open the store
// This reads the acl file and subscribes to file changes.
func (aclStore *AclFileStore) Open() (err error) {
	slog.Info("AclFileStore.Open", "serviceID", aclStore.serviceID)

	// create the acl store folder if it doesn't exist
	storeFolder := path.Dir(aclStore.storePath)
	if _, err2 := os.Stat(storeFolder); os.IsNotExist(err2) {
		slog.Info("Creating store directory", "directory", storeFolder)
		err = os.MkdirAll(storeFolder, 0700)
	}
	if err != nil {
		return
	}

	// create a new file if it doesn't exist
	if _, err2 := os.Stat(aclStore.storePath); os.IsNotExist(err2) {
		file, err := os.OpenFile(aclStore.storePath, os.O_RDWR|os.O_CREATE, 0600)
		if err == nil {
			file.Close()
		} else {
			return err
		}
	}

	err = aclStore.Reload()
	if err != nil {
		return err
	}
	// watcher handles debounce of too many events
	//aclStore.watcher, err = watcher.WatchFile(ctx, aclStore.storePath, aclStore.Reload)
	return err
}

// Reload the ACL store from file
func (aclStore *AclFileStore) Reload() error {
	slog.Info("AclFileStore.Reload", "serviceID", aclStore.serviceID, "store", aclStore.storePath)

	raw, err := os.ReadFile(aclStore.storePath)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	err = yaml.Unmarshal(raw, &aclStore.groups)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// build the role map index for each user in the groups
	slog.Info("AclFileStore.Reload complete",
		"serviceID", aclStore.serviceID, "#groups", len(aclStore.groups))
	aclStore.userGroupRoles = aclStore._buildUserGroupRoles()

	return nil
}

// RemoveSource removes a source from a group
// If the source doesn't exist then returns without error
// The caller must be an administrator or service.
//
//	publisherID is required and identifies the publisher of the event
//	thingID is optional and identifies a specific Thing, "" for all things of the publisher
func (aclStore *AclFileStore) RemoveSource(publisherID, thingID string, groupID string) error {
	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	group, found := aclStore.groups[groupID]
	if !found {
		return fmt.Errorf("group '%s' does not exist", groupID)
	}
	for i, source := range group.Sources {
		if source.PublisherID == publisherID && source.ThingID == thingID {
			group.Sources = append(group.Sources[:i], group.Sources[i+1:]...)
		}
	}
	aclStore.groups[groupID] = group
	err := aclStore.Save()
	return err
}

// RemoveUser removes a user from a group and update the store.
//
//	userID user to remove
//	groupID  group to remove from
//
// returns an error if the group doesn't exist or saving the update fails
func (aclStore *AclFileStore) RemoveUser(userID string, groupID string) error {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	group, found := aclStore.groups[groupID]
	if !found {
		return fmt.Errorf("group '%s' does not exist", groupID)
	}
	delete(group.MemberRoles, userID)

	// remove the group also from the client index
	groupRoles, found := aclStore.userGroupRoles[userID]
	if found {
		delete(groupRoles, groupID)
	}

	err := aclStore.Save()
	return err
}

// RemoveUserAll removes a user from all groups and update the store.
//
//	clientID client to remove
//
// this succeeds unless the store cannot be written
func (aclStore *AclFileStore) RemoveUserAll(userID string) error {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	// remove the client from each individual group
	groupRoles := aclStore.userGroupRoles[userID]
	for groupID, _ := range groupRoles {
		aclGroup, found := aclStore.groups[groupID]
		if found {
			delete(aclGroup.MemberRoles, userID)
		}
	}

	// remove all the group roles of the client
	delete(aclStore.userGroupRoles, userID)

	err := aclStore.Save()
	return err
}

// setUserRole sets a user's role in a group and update the store.
// This add/updates the client's role, saves it to a temp file and move the result to the store file.
//
//	userID   user to assign the role
//	role     one of UserRoleViewer, UserRoleOperator, UserRoleManager or GroupRoleNone to remove the role
//	groupID  group where the role applies
func (aclStore *AclFileStore) setUserRole(userID string, role string, groupID string) (err error) {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	// update the group
	aclGroup, found := aclStore.groups[groupID]
	if !found {
		aclGroup = authz.Group{
			ID:          groupID,
			MemberRoles: authz.UserRoleMap{},
			Sources:     make([]authz.EventSource, 0),
		}
		aclStore.groups[groupID] = aclGroup
	}
	aclGroup.MemberRoles[userID] = role

	// update the index
	groupRoles, found := aclStore.userGroupRoles[userID]
	if !found {
		groupRoles = make(authz.UserRoleMap)
		aclStore.userGroupRoles[userID] = groupRoles
	}
	groupRoles[groupID] = role
	aclStore.groups[groupID] = aclGroup

	// save
	err = aclStore.Save()
	return err
}

// SetUserRole sets a user ACL and update the store.
// This updates the user's role, saves it to a temp file and move the result to the store file.
//
//	userID   client to assign the role
//	role     one of UserRoleViewer, UserRoleOperator, UserRoleManager, or UserRoleService
//	groupID  group where the role applies
func (aclStore *AclFileStore) SetUserRole(userID string, role string, groupID string) (err error) {
	isUserRole := role == authz.UserRoleViewer ||
		role == authz.UserRoleManager ||
		role == authz.UserRoleOperator
	if !isUserRole {
		return fmt.Errorf("role '%s' doesn't apply to users", role)
	}
	return aclStore.setUserRole(userID, role, groupID)
}

// Save the store to file
// Save is not concurrent safe and intended for use by setRole and removeClient which are.
func (aclStore *AclFileStore) Save() error {
	var file *os.File
	var tempFileName string
	var err error

	yamlData, err := yaml.Marshal(aclStore.groups)
	if err == nil {
		folder := path.Dir(aclStore.storePath)
		file, err = os.CreateTemp(folder, "authz-aclfilestore")
	}
	// file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		tempFileName = file.Name()

		writer := bufio.NewWriter(file)
		_, err = writer.Write(yamlData)
		_ = writer.Flush()
		file.Close()
	}

	if err == nil {
		err = os.Rename(tempFileName, aclStore.storePath)
	}
	if err != nil {
		err := fmt.Errorf("failed saving ACL store: %s", err)
		slog.Error(err.Error())
		return err
	}
	return nil
}

// NewAuthzFileStore creates an instance of a file based authz store
//
//	filepath is the location of the store file. See also DefaultAclFilename for the recommended name.
func NewAuthzFileStore(filepath string) *AclFileStore {
	store := &AclFileStore{
		serviceID:      authz.AuthzServiceName,
		groups:         make(map[string]authz.Group),
		userGroupRoles: make(map[string]authz.UserRoleMap),

		storePath: filepath,
	}
	return store
}
