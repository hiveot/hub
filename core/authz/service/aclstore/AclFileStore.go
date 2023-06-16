package aclstore

import (
	"bufio"
	"fmt"
	"github.com/hiveot/hub/core/authz"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync"

	"gopkg.in/yaml.v3"
)

// AclFileStore is an in-memory ACL store based on the state store
type AclFileStore struct {
	serviceID string

	// Groups is an index of ACL groups by their name. Stored
	// map[groupName]Group
	groups map[string]authz.Group `yaml:"groups"`

	// state store
	//store     state.IClientState
	storePath string

	// index of clients and their group roles. Updated on load.
	// intended for fast lookup of roles
	//map[clientID]map[groupName]role
	clientGroupRoles map[string]authz.RoleMap
	mutex            sync.RWMutex
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

// GetGroup returns the group of the given name
func (aclStore *AclFileStore) GetGroup(groupName string) (authz.Group, error) {

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()
	group, found := aclStore.groups[groupName]
	if !found {
		err := fmt.Errorf("group '%s' does not exist", groupName)
		return group, err
	}
	return group, nil
}

// GetGroupRoles returns the roles a thing or user has in various groups
func (aclStore *AclFileStore) GetGroupRoles(clientID string) authz.RoleMap {

	aclStore.mutex.RLock()
	defer aclStore.mutex.RUnlock()

	// change map of group->clientrole to list
	roles := aclStore.clientGroupRoles[clientID]

	return roles
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
	for groupName := range groupRoles {
		// client is a member of this group, check if the thingID is also a member
		_, thingIsMember := aclStore.clientGroupRoles[thingID]
		// all things are a member of the all group
		if thingIsMember || groupName == authz.AllGroupName {
			sharedGroups = append(sharedGroups, groupName)
		}
	}
	return sharedGroups
}

// GetRole returns the highest role of a user has in groups shared with the thingID
// Intended to get client permissions in case of overlapping groups
func (aclStore *AclFileStore) GetRole(clientID string, thingID string) string {
	groups := aclStore.GetSharedGroups(clientID, thingID)
	highestRole := aclStore.GetHighestRole(clientID, groups)
	return highestRole
}

// GetHighestRole returns the highest role of a user has in a list of group
// Intended to get client permissions in case of overlapping groups
func (aclStore *AclFileStore) GetHighestRole(clientID string, groupIDs []string) string {
	highestRole := authz.ClientRoleNone

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

// IsRoleGreaterEqual returns true if a user role has same or greater permissions
// than the minimum role.
func IsRoleGreaterEqual(role string, minRole string) bool {
	if minRole == authz.ClientRoleNone || role == minRole {
		return true
	}
	if minRole == authz.ClientRoleViewer && role != authz.ClientRoleNone {
		return true
	}
	if minRole == authz.ClientRoleOperator && (role == authz.ClientRoleManager) {
		return true
	}
	return false
}

// ListGroups return ... a list of all groups
// TODO: apply limit and offset.
func (aclStore *AclFileStore) ListGroups(limit int, offset int) []authz.Group {
	groups := make([]authz.Group, 0, len(aclStore.groups))
	for _, group := range aclStore.groups {
		groups = append(groups, group)
	}
	return groups
}

// Open the store
// This reads the acl file and subscribes to file changes.
func (aclStore *AclFileStore) Open() error {
	slog.Info("AclFileStore.Open", "serviceID", aclStore.serviceID)

	// create the acl store folder if it doesn't exist
	storeFolder := path.Dir(aclStore.storePath)
	if _, err := os.Stat(storeFolder); os.IsNotExist(err) {
		slog.Info("Creating store directory", "directory", storeFolder)
		_ = os.MkdirAll(storeFolder, 0700)
	}

	// create a new file if it doesn't exist
	if _, err := os.Stat(aclStore.storePath); os.IsNotExist(err) {
		file, err := os.OpenFile(aclStore.storePath, os.O_RDWR|os.O_CREATE, 0600)
		if err == nil {
			file.Close()
		}
	}

	err := aclStore.Reload()
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
	for groupName, group := range aclStore.groups {
		// iterate the group members and add them to the client index along with its group role
		for memberID, memberRole := range group.MemberRoles {
			//
			groupRoles, found := clientGroupRoles[memberID]
			if !found {
				// Need to add the set of group roles for this client
				groupRoles = make(authz.RoleMap)
				clientGroupRoles[memberID] = groupRoles
			}
			groupRoles[groupName] = memberRole
		}
	}
	slog.Info("AclFileStore.Reload complete", "serviceID", aclStore.serviceID, "groups", len(clientGroupRoles))
	aclStore.clientGroupRoles = clientGroupRoles
	return nil
}

// Remove removes a client from a group and update the store.
//
//	serviceID login name to assign the role
//	groupID  group where the role applies.
//
// returns an error if the group doesn't exist or saving the update fails
func (aclStore *AclFileStore) Remove(clientID string, groupID string) error {

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

// RemoveAll removes a client from all groups and update the store.
//
//	clientID user or thingID to remove from all groups
func (aclStore *AclFileStore) RemoveAll(clientID string) error {

	for key := range aclStore.groups {
		aclStore.Remove(clientID, key)
	}
	return nil
}

// SetRole sets a user ACL and update the store.
// This updates the user's role, saves it to a temp file and move the result to the store file.
//
//	clientID   client to assign the role
//	groupName  group where the role applies
//	role     one of ClientRoleViewer, ClientRoleOperator, ClientRoleManager, ClientRoleThing or ClientRoleNone to remove the role
func (aclStore *AclFileStore) SetRole(clientID string, groupName string, role string) error {

	// Prevent concurrently running Reload and SetRole
	aclStore.mutex.Lock()
	defer aclStore.mutex.Unlock()

	// update the group
	aclGroup, found := aclStore.groups[groupName]
	if !found {
		aclGroup = authz.NewGroup(groupName)
		aclStore.groups[groupName] = aclGroup
	}
	aclGroup.MemberRoles[clientID] = role

	// update the index
	groupRoles, found := aclStore.clientGroupRoles[clientID]
	if !found {
		groupRoles = make(authz.RoleMap)
		aclStore.clientGroupRoles[clientID] = groupRoles
	}
	groupRoles[groupName] = role

	// save
	err := aclStore.Save()

	slog.Info("AclFileStore.SetRole", "serviceID", clientID, "role", role, "group", groupName)
	return err
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

// NewAclFileStore creates an instance of a file based ACL store
//
//	filepath is the location of the store. See also DefaultAclFilename for the recommended name.
//	serviceID is for logging which authservice is accessing it
func NewAclFileStore(filepath string, serviceID string) *AclFileStore {
	store := &AclFileStore{
		serviceID:        serviceID,
		groups:           make(map[string]authz.Group),
		clientGroupRoles: make(map[string]authz.RoleMap),

		storePath: filepath,
	}
	return store
}
