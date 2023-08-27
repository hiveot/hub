package authz

import "time"

// AuthzServiceName default name of the service, used for logging and identification
const AuthzServiceName = "authz"
const DefaultAclFilename = "authz-groups.acl"

// User roles set permissions for operations on Things that are members of the same group
// The mapping of roles to operations is currently hard coded aimed at managing Things
const (
	// UserRoleNone indicates that the client has no particular role. It can not do anything until
	// the role is upgraded to viewer or better.
	//  Read permissions: none
	//  Write permissions: none
	UserRoleNone = "none"

	// UserRoleManager lets a client subscribe to Thing TD, events, publish actions and update configuration
	//  Read permissions: readEvents
	//  Write permissions: pubActions
	UserRoleManager = "manager"

	// UserRoleOperator lets a client subscribe to Thing TD, events and publish actions
	//  Read permissions: readEvents, readActions
	//  Write permissions: pubActions
	UserRoleOperator = "operator"

	// UserRoleService identifies the client as a service
	// Services can subscribe to and publish actions and events
	//  Read permissions: readActions, readEvents
	//  Write permissions: pubEvents, pubActions
	UserRoleService = "service"

	// UserRoleViewer lets a client subscribe to Thing TD and Thing Events
	//  Read permissions: readTDs, readEvents
	//  Write permissions: none
	UserRoleViewer = "viewer"
)

// Permissions that can be authorized
// The list of permissions is currently hard coded aimed at managing Things
// It is expected that future services will add permissions but that is for later.
//const (
//	// PermPubActions permission of publishing actions
//	PermPubActions = "permPubActions"
//
//	// PermPubEvents permission to publish events, including property value events
//	PermPubEvents = "permPubEvents"
//
//	// PermReadActions permission of read/subscribe to actions
//	PermReadActions = "permReadActions"
//
//	// PermReadEvents permission to read/subscribe to events
//	PermReadEvents = "permReadEvents"
//)

// predefined group names
const (
	// AllGroupID is the built-in group containing all resources
	AllGroupID = "all"
)

// DefaultGroupRetention default group data retention in seconds is 7 days
const DefaultGroupRetention = 3600 * 24 * 7

// UserRoleMap with userID:role for all users is a group
type UserRoleMap map[string]string

// GroupRoleMap with groupID:role for all groups of a user
type GroupRoleMap map[string]string

// EventSource defines the publisher (device or service) and Thing that generates events
type EventSource struct {
	PublisherID string
	ThingID     string
}

// Group containing a list of sources and consumers
type Group struct {
	// Unique ID of the group. This is immutable.
	ID string
	// Group presentation name
	DisplayName string

	// Sources contains the Publisher/Things that are event sources of the group
	Sources []EventSource

	// MemberRoles is a map of users (incl services) and their role in this group
	MemberRoles UserRoleMap

	// data retention period in seconds
	Retention time.Duration
}

// ManageAuthzCapability is the thingID of the capability that handles authorization management
const ManageAuthzCapability = "manage"

// Authorization management request/response messages
// This requires administrator permissions.

// AddSourceAction defines the action to add a Thing to a group
const AddSourceAction = "addSource"

// AddSourceReq request message to add an event source to a group
// The caller must be an administrator or service.
type AddSourceReq struct {
	// name of existing group
	GroupID string `json:"groupID"`
	// Publisher of the event
	PublisherID string `json:"publisherID"`
	// the thingID to add in the IoT device role or "" to include all things
	ThingID string `json:"thingID,omitempty"`
}

// AddUserAction defines the action to add a user to a group
const AddUserAction = "addUser"

// AddUserReq request message to add a User to a group
// The caller must be an administrator or service.
type AddUserReq struct {
	// name of existing group
	GroupID string `json:"groupID"`
	// user to add
	UserID string `json:"userID"`
	// role the user is added in: viewer, operator, manager or service
	Role string `json:"role"`
}

// CreateGroupAction defines the action to add a group with a retention period
const CreateGroupAction = "createGroup"

// CreateGroupReq request message to add a group.
// The caller must be an administrator or service.
type CreateGroupReq struct {
	// unique name of the group
	GroupID string `json:"groupID"`
	// presentation name
	DisplayName string `json:"displayName"`
	// retention period in seconds of events in this group
	// use 0 for indefinitely
	Retention uint64 `json:"retention"`
}

// DeleteGroupAction defines the action to delete the group with the given name
const DeleteGroupAction = "deleteGroup"

// DeleteGroupReq request message to delete a group and its archived messages
// The caller must be an administrator or service.
type DeleteGroupReq struct {
	// group to delete
	GroupID string `json:"groupID"`
}

// GetGroupAction defines the action to request the content of a group
const GetGroupAction = "getGroup"

// GetGroupReq requests the group and its members
// The caller must be an administrator or service.
type GetGroupReq struct {
	GroupID string `json:"groupID"`
}

// GetGroupResp returns the group and its members
type GetGroupResp struct {
	Group Group `json:"group"`
}

//// GetPermissionsAction defines the action to get the permissions a client has on one or more things
//const GetPermissionsAction = "getPermissions"

//// GetPermissionsReq requests the permissions a user has for one or more Things
//// Intended to determine what operations a user can perform on Things.
//// Administrator or service can use any client ID.
//// Other callers can only request their own permissions.
//type GetPermissionsReq struct {
//	// Client whose permissions to obtain
//	UserID string `json:"clientID"`
//	// List of things whose permissions to get
//	ThingIDs []string `json:"thingID"`
//}
//
//// GetPermissionsResp returns a list of permissions the client has on a thing
//// Returns a map of permissions by thingID, eg PermPubAction, etc
//type GetPermissionsResp struct {
//	// List of permissions for each requested thing
//	Permissions map[string][]string `json:"permissions"`
//}

//// GetRoleAction defines the action to request the role of a client for a Thing
//const GetRoleAction = "getRole"
//
//// GetRoleReq request message to get the role of a client for a thing
//type GetRoleReq struct {
//	UserID string `json:"clientID"`
//	ThingID  string `json:"thingID"`
//}
//
//// GetRoleResp response with the role
//type GetRoleResp struct {
//	Role string `json:"role"`
//}

// GetUserGroupsAction defines the action to list defined groups
const GetUserGroupsAction = "getUserGroups"

// GetUserGroupsReq requests a list of groups available to the client
// Administrators and services can provide a clientID other than their own,
// or use "" for clientID to get all groups
// Any other user must supply their own clientID
type GetUserGroupsReq struct {
	UserID string `json:"userID"`
}

// GetUserGroupsResp returns a list of groups
type GetUserGroupsResp struct {
	Groups []Group `json:"groups"`
}

// GetUserRolesAction defines the action to request a map of [group]role of a user
const GetUserRolesAction = "getUserRoles"

// GetUserRolesReq request message to get a map of group-roles for all groups the user is a member of.
type GetUserRolesReq struct {
	UserID string `json:"userID"`
}

// GetUserRolesResp response with a list of groupIDs and the user's role in that group
type GetUserRolesResp struct {
	Roles UserRoleMap `json:"roles"` // map of role by groupID
}

// RemoveSourceAction defines the action to remove a source Thing from a group
const RemoveSourceAction = "removeSource"

// RemoveSourceReq requests removal of a device/thing or service/capability source
// from a group. The caller must be an administrator or service.
type RemoveSourceReq struct {
	PublisherID string `json:"publisherID"`       // Service/Thing(s) publisher
	ThingID     string `json:"thingID,omitempty"` // optional a specific Thing/Capability
	GroupID     string `json:"groupID"`           // Group to remove the thing(s) from
}

// RemoveUserAction defines the action to remove a client from a group
const RemoveUserAction = "removeUser"

// RemoveUserReq requests removal of a user from a group
// The caller must be an administrator or service.
type RemoveUserReq struct {
	UserID  string `json:"userID"`
	GroupID string `json:"groupID,omitempty"`
}

// RemoveUserAllAction defines the action to remove a client from all groups
const RemoveUserAllAction = "removeUserAll"

// RemoveUserAllReq requests removal of a client from all groups
// The caller must be an administrator or service.
type RemoveUserAllReq struct {
	UserID string `json:"userID"`
}

// SetUserRoleAction updates the role of a user in a group
const SetUserRoleAction = "setClientRole"

// SetUserRoleReq requests update of a user role in a group
// If the user is not a member of a group, the user will be added.
// The role must be one of the user roles.
type SetUserRoleReq struct {
	UserID   string `json:"userID"`
	GroupID  string `json:"groupID"`
	UserRole string `json:"userRole"`
}

// IAuthz defines the management capabilities of the authorization service
type IAuthz interface {

	// AddSource adds an event source with the thing role to a group
	//  publisherID is the device or service that publishes the events
	//  thingID is the Thing whose info is published or "" for all things of the publisher
	AddSource(publisherID string, thingID string, groupID string) error

	// AddUser adds a consumer to a group with the user role:
	//  manager, operator viewer or service.
	// If the client is already in the group this returns without error
	// See ClientRole...
	AddUser(userID string, role string, groupID string) error

	// CreateGroup creates a new group
	// If the group exists this returns without error
	// Use retention 0 to retain messages indefinitely
	//
	//	groupID unique ID of the group
	//  displayName of the group
	//	retention period of events in this group. 0 for indefinite
	CreateGroup(groupID string, displayName string, retention time.Duration) error

	// DeleteGroup deletes the group and all its resources. This is not recoverable.
	// If the client doesn't exist then returns without error
	DeleteGroup(groupID string) error

	// GetGroup returns the group with the given name, or an error if group is not found.
	// groupID must not be empty and must be an existing group
	// Returns an error if the group does not exist.
	GetGroup(groupID string) (group Group, err error)

	// GetPermissions returns the permissions the client has for Things.
	// clientID is optional. The default is to use the connecting client's ID.
	// Only managers and services are allowed to choose a clientID different from their own.
	// Returns an map of permissions for each thing, eg PermEmitAction, etc
	//GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error)

	// GetRole determines the highest role a user has for a thing
	// If the client is a member of multiple groups each group role is checked.
	//GetRole(clientID string, thingID string) (role string, err error)

	// GetUserGroups returns the list of groups the user is a member of
	// If userID is "" then all groups are returned.
	GetUserGroups(userID string) (groups []Group, err error)

	// GetUserRoles returns a map of [groupID]role for groups the user is a member of.
	GetUserRoles(userID string) (roles UserRoleMap, err error)

	// RemoveSource removes a source from a group
	// If the source doesn't exist then returns without error
	// The caller must be an administrator or service.
	//  publisherID is required and identifies the publisher of the event
	//  thingID is optional and identifies a specific Thing, "" for all things of the publisher
	RemoveSource(publisherID, thingID string, groupID string) error

	// RemoveUser removes a user from a group
	// If the user doesn't exist then returns without error
	// The caller must be an administrator or service.
	RemoveUser(userID string, groupID string) error

	// RemoveUserAll removes a user from all groups.
	// If the user doesn't exist then returns without error
	// The caller must be an administrator or service.
	RemoveUserAll(userID string) error

	// SetUserRole changes the role for the user in a group.
	//
	// If the user is not a member of a group the user will be added.
	// The role must be one of the user roles viewer, operator, manager, or service
	SetUserRole(userID string, userRole string, groupID string) error

	// Start the service, open the store, starts listening for action requests
	Start() error

	// Stop the service. close connections
	Stop()
}
