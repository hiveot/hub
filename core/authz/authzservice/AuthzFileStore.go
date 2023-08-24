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

	// index of clients and their group roles. Updated on load.
	// intended for fast lookup of roles
	//map[clientID]map[groupID]role
	clientGroupRoles map[string]authz.RoleMap
	mutex            sync.RWMutex
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
		MemberRoles: authz.RoleMap{},
		Retention:   retention,
	}
	err := aclStore.Save()
	return err
}

// AddService adds a client with the service role to a group
func (aclStore *AclFileStore) AddService(serviceID string, groupID string) error {
	err := aclStore.setRole(serviceID, authz.GroupRoleService, groupID)
	return err
}

// AddThing adds a client with the thing role to a group
func (aclStore *AclFileStore) AddThing(thingID string, groupID string) error {
	err := aclStore.setRole(thingID, authz.GroupRoleThing, groupID)
	return err
}

// AddUser adds a client with the user role to a group
func (aclStore *AclFileStore) AddUser(userID string, role string, groupID string) (err error) {

	if role == authz.GroupRoleViewer ||
		role == authz.GroupRoleManager ||
		role == authz.GroupRoleOperator {

		err = aclStore.setRole(userID, role, groupID)
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
	err := aclStore.Save()

	slog.Info("group removed", "groupID", groupID)
	return err
}

// GetClientRoles returns the roles a thing or user has in various groups
func (aclStore *AclFileStore) GetClientRoles(clientID string) (authz.RoleMap, error) {

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	roles := aclStore.clientGroupRoles[clientID]

	return roles, nil
}

// GetClientsGroupsRole returns the roles clients have in each group
// This returns a map of clients with their groups and role in that group
func (aclStore *AclFileStore) GetClientsGroupsRole() map[string]authz.RoleMap {
	shallowCopy := make(map[string]authz.RoleMap)
	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	aclStore.mutex.RLock()
	for clientID, roles := range aclStore.clientGroupRoles {
		shallowCopy[clientID] = roles
	}
	aclStore.mutex.RUnlock()

	return shallowCopy
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

// GetHighestRole returns the highest role of a user has in a list of group
// Intended to get client permissions in case of overlapping groups
func (aclStore *AclFileStore) GetHighestRole(clientID string, groupIDs []string) string {
	highestRole := authz.GroupRoleNone

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	groupRoles := aclStore.clientGroupRoles[clientID]

	for _, groupID := range groupIDs {
		clientRole, found := groupRoles[groupID]
		if found && IsRoleGreaterEqual(clientRole, highestRole) {
			highestRole = clientRole
		}
	}
	return highestRole
}

// GetPermissions returns a list of permissions a client has for Things
func (aclStore *AclFileStore) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
	permissions = make(map[string][]string)
	for _, thingID := range thingIDs {
		var thingPerm []string
		clientRole, _ := aclStore.GetRole(clientID, thingID)
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
func (aclStore *AclFileStore) GetRole(clientID string, thingID string) (string, error) {
	groups := aclStore.GetSharedGroups(clientID, thingID)
	highestRole := aclStore.GetHighestRole(clientID, groups)
	return highestRole, nil
}

// GetSharedGroups returns a list of groups both a client and thing are a member of.
func (aclStore *AclFileStore) GetSharedGroups(clientID string, thingID string) []string {
	sharedGroups := []string{}

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()
	// iterate the groups of the client
	groupRoles, clientHasMemberships := aclStore.clientGroupRoles[clientID]
	if !clientHasMemberships {
		return sharedGroups
	}
	for groupID := range groupRoles {
		// client is a member of this group, check if the thingID is also a member
		_, thingIsMember := aclStore.clientGroupRoles[thingID]
		// all things are a member of the all group
		if thingIsMember || groupID == authz.AllGroupID {
			sharedGroups = append(sharedGroups, groupID)
		}
	}
	return sharedGroups
}

// IsRoleGreaterEqual returns true if a user role has same or greater permissions
// than the minimum role.
func IsRoleGreaterEqual(role string, minRole string) bool {
	if minRole == authz.GroupRoleNone || role == minRole {
		return true
	}
	if minRole == authz.GroupRoleViewer && role != authz.GroupRoleNone {
		return true
	}
	if minRole == authz.GroupRoleOperator && (role == authz.GroupRoleManager) {
		return true
	}
	return false
}

// ListGroups return ... a list of all groups available to the client
//
//	clientID client that is a member of the groups to return, or "" for all groups
func (aclStore *AclFileStore) ListGroups(clientID string) ([]authz.Group, error) {
	groups := make([]authz.Group, 0, len(aclStore.groups))
	for _, group := range aclStore.groups {
		_, found := group.MemberRoles[clientID]
		if found || clientID == "" {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

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

	// build the client index for each client in the groups
	clientGroupRoles := make(map[string]authz.RoleMap)
	// for each group, add its members to the client index
	for groupID, group := range aclStore.groups {
		// iterate the group members and add them to the client index along with its group role
		for memberID, memberRole := range group.MemberRoles {
			//
			groupRoles, found := clientGroupRoles[memberID]
			if !found {
				// Need to add the set of group roles for this client
				groupRoles = make(authz.RoleMap)
				clientGroupRoles[memberID] = groupRoles
			}
			groupRoles[groupID] = memberRole
		}
	}
	slog.Info("AclFileStore.Reload complete", "serviceID", aclStore.serviceID, "groups", len(clientGroupRoles))
	aclStore.clientGroupRoles = clientGroupRoles
	return nil
}

// RemoveClient removes a client from a group and update the store.
//
//	serviceID login name to assign the role
//	groupID  group where the role applies.
//
// returns an error if the group doesn't exist or saving the update fails
func (aclStore *AclFileStore) RemoveClient(clientID string, groupID string) error {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	aclGroup, found := aclStore.groups[groupID]
	if !found {
		return fmt.Errorf("group '%s' does not exist", groupID)
	}
	delete(aclGroup.MemberRoles, clientID)

	// remove the group also from from the client index
	groupRoles, found := aclStore.clientGroupRoles[clientID]
	if found {
		delete(groupRoles, groupID)
	}

	err := aclStore.Save()

	slog.Info("client removed from group", "clientID", clientID, "groupID", groupID)
	return err
}

// RemoveClientAll removes a client from all groups and update the store.
//
//	clientID client to remove
//
// this succeeds unless the store cannot be written
func (aclStore *AclFileStore) RemoveClientAll(clientID string) error {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	// remove the client from each individual group
	groupRoles := aclStore.clientGroupRoles[clientID]
	for groupID, _ := range groupRoles {
		aclGroup, found := aclStore.groups[groupID]
		if found {
			delete(aclGroup.MemberRoles, clientID)
		}
	}

	// remove all the group roles of the client
	delete(aclStore.clientGroupRoles, clientID)

	err := aclStore.Save()

	slog.Info("client removed from all groups", "clientID", clientID)
	return err
}

// setRole sets a client ACL and update the store.
// This add/updates the client's role, saves it to a temp file and move the result to the store file.
//
//	clientID   client to assign the role
//	role       one of GroupRoleViewer, GroupRoleOperator, GroupRoleManager, GroupRoleThing or GroupRoleNone to remove the role
//	groupID  group where the role applies
func (aclStore *AclFileStore) setRole(clientID string, role string, groupID string) (err error) {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	// update the group
	aclGroup, found := aclStore.groups[groupID]
	if !found {
		aclGroup = authz.Group{
			ID:          groupID,
			MemberRoles: authz.RoleMap{},
		}
		aclStore.groups[groupID] = aclGroup
	}
	aclGroup.MemberRoles[clientID] = role

	// update the index
	groupRoles, found := aclStore.clientGroupRoles[clientID]
	if !found {
		groupRoles = make(authz.RoleMap)
		aclStore.clientGroupRoles[clientID] = groupRoles
	}
	groupRoles[groupID] = role

	// save
	err = aclStore.Save()

	slog.Info("AclFileStore.SetRole", "clientID", clientID, "role", role, "group", groupID)
	return err
}

// SetUserRole sets a user ACL and update the store.
// This updates the user's role, saves it to a temp file and move the result to the store file.
//
//	userID   client to assign the role
//	role     one of GroupRoleViewer, GroupRoleOperator, GroupRoleManager, GroupRoleThing or GroupRoleNone to remove the role
//	groupID  group where the role applies
func (aclStore *AclFileStore) SetUserRole(userID string, role string, groupID string) (err error) {
	isUserRole := role == authz.GroupRoleViewer ||
		role == authz.GroupRoleManager ||
		role == authz.GroupRoleOperator
	if !isUserRole {
		return fmt.Errorf("role '%s' doesn't apply to users", role)
	}
	return aclStore.setRole(userID, role, groupID)
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
		serviceID:        authz.AuthzServiceName,
		groups:           make(map[string]authz.Group),
		clientGroupRoles: make(map[string]authz.RoleMap),

		storePath: filepath,
	}
	return store
}
