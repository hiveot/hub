package authz

// AuthzServiceName default name of the service, used for logging and identification
const AuthzServiceName = "authz"

// Client roles set permissions for operations on Things that are members of the same group
// The mapping of roles to operations is currently hard coded aimed at managing Things
const (
	// ClientRoleNone indicates that the client has no particular role. It can not do anything until
	// the role is upgraded to viewer or better.
	//  Read permissions: none
	//  Write permissions: none
	ClientRoleNone = "none"

	// ClientRoleIotDevice for IoT devices that read/write for things it is the publisher of.
	// IoT Devices can publish events and updates for Things it the publisher of. This is determined
	// by the deviceID that is included in the thingID.
	//  Read permissions: readActions
	//  Write permissions: pubEvents, pubActions
	ClientRoleIotDevice = "iotdevice"

	// ClientRoleManager lets a client subscribe to Thing TD, events, publish actions and update configuration
	//  Read permissions: readEvents
	//  Write permissions: pubActions
	ClientRoleManager = "manager"

	// ClientRoleOperator lets a client subscribe to Thing TD, events and publish actions
	//  Read permissions: readEvents, readActions
	//  Write permissions: pubActions
	ClientRoleOperator = "operator"

	// ClientRoleService identifies the client as a service
	// Services can subscribe to and publish actions and events
	//  Read permissions: readActions, readEvents
	//  Write permissions: pubEvents, pubActions
	ClientRoleService = "service"

	// ClientRoleThing identifies the client as a Thing
	// Things can publish events and updates for themselves.
	//  Read permissions: readAction
	//  Write permissions: pubEvents, pubActions
	ClientRoleThing = "thing"

	// ClientRoleViewer lets a client subscribe to Thing TD and Thing Events
	//  Read permissions: readTDs, readEvents
	//  Write permissions: none
	ClientRoleViewer = "viewer"
)

// Permissions that can be authorized
// The list of permissions is currently hard coded aimed at managing Things
// It is expected that future services will add permissions but that is for later.
const (
	// PermPubActions permission of publishing actions
	PermPubActions = "permPubActions"

	// PermPubEvents permission to publish events, including property value events
	PermPubEvents = "permPubEvents"

	// PermReadActions permission of read/subscribe to actions
	PermReadActions = "permReadActions"

	// PermReadEvents permission to read/subscribe to events
	PermReadEvents = "permReadEvents"
)

// predefined group names
const (
	// NoAuthGroupName is the built-in group for unauthenticated clients
	NoAuthGroupName = "unauthenticated"

	// AllGroupName is the built-in group containing all resources
	AllGroupName = "all"
)

// RoleMap for members or memberships
type RoleMap map[string]string // clientID:role, groupName:role

// Group is a map of clientID:role
type Group struct {
	Name string
	// map of clients and their role in this group
	MemberRoles RoleMap
}

//// NewGroup creates an instance of a group with member roles
//func NewGroup(groupName string) Group {
//	return Group{
//		Name:        groupName,
//		MemberRoles: make(RoleMap),
//	}
//}

// list of actions and their payload

// ManageAuthzCapability is the thingID of the capability that handles authorization management
const ManageAuthzCapability = "manage"

// Authorization management request/response messages
// This requires administrator permissions.

// AddGroupAction defines the action to add a group with a retention period
const AddGroupAction = "addGroup"

// AddGroupReq request message to add a group.
// The caller must be an administrator or service.
type AddGroupReq struct {
	// unique name of the group
	GroupName string `json:"groupName"`
	// retention period in seconds of events in this group
	Retention uint64 `json:"retention"`
}

// AddThingAction defines the action to add a Thing to a group
const AddThingAction = "addThing"

// AddThingReq request message to add a Thing to a group
// The caller must be an administrator or service.
type AddThingReq struct {
	// name of existing group
	GroupName string `json:"groupName"`
	// the thingID to add in the IoT device role
	ThingID string `json:"thingID"`
}

// AddServiceAction defines the action to add a service to a group
const AddServiceAction = "addService"

// AddServiceReq request message to add a Service to a group
// The caller must be an administrator or service.
type AddServiceReq struct {
	// name of existing group
	GroupName string `json:"groupName"`
	// serviceID to add in the service role
	ServiceID string `json:"serviceID"`
}

// AddUserAction defines the action to add a user to a group
const AddUserAction = "addUser"

// AddUserReq request message to add a User to a group
// The caller must be an administrator or service.
type AddUserReq struct {
	// name of existing group
	GroupName string `json:"groupName"`
	// user to add
	UserID string `json:"userID"`
	// role the user is added in: viewer, operator or manager
	Role string `json:"role"`
}

// DeleteGroupAction defines the action to delete the group with the given name
const DeleteGroupAction = "deleteGroup"

// DeleteGroupReq request message to delete a group and its archived messages
// The caller must be an administrator or service.
type DeleteGroupReq struct {
	// group to delete
	GroupName string `json:"groupName"`
}

// GetClientRolesAction defines the action to request a list of clients and their roles of a group
const GetClientRolesAction = "getClientRoles"

// GetClientRolesReq request message to get a list group and roles of a client
type GetClientRolesReq struct {
	ClientID string `json:"clientID"`
}

// GetClientRolesResp response with a list of groups and the client's roles
type GetClientRolesResp struct {
	Roles RoleMap `json:"roles"` // map of role by groupID
}

// GetGroupAction defines the action to request the content of a group
const GetGroupAction = "getGroup"

// GetGroupReq requests the group and its members
// The caller must be an administrator or service.
type GetGroupReq struct {
	GroupName string `json:"groupName"`
}

// GetGroupResp returns the group and its members
type GetGroupResp struct {
	Group Group `json:"group"`
}

// GetPermissionsAction defines the action to get the permissions a client has on one or more things
const GetPermissionsAction = "getPermissions"

// GetPermissionsReq requests the permissions a user has for one or more Things
// Intended to determine what operations a user can perform on Things.
// Administrator or service can use any client ID.
// Other callers can only request their own permissions.
type GetPermissionsReq struct {
	// Client whose permissions to obtain
	ClientID string `json:"clientID"`
	// List of things whose permissions to get
	ThingIDs []string `json:"thingID"`
}

// GetPermissionsResp returns a list of permissions the client has on a thing
// Returns a map of permissions by thingID, eg PermPubAction, etc
type GetPermissionsResp struct {
	// List of permissions for each requested thing
	Permissions map[string][]string `json:"permissions"`
}

// ListGroupsAction defines the action to list defined groups
const ListGroupsAction = "listGroups"

// ListGroupsReq requests a list of groups available to the client
// Administrators and services can provide a clientID other than their own,
// or use "" for clientID to get all groups
// Any other user must supply their own clientID
type ListGroupsReq struct {
	ClientID string `json:"clientID"`
}

// ListGroupsResp returns a list of groups
type ListGroupsResp struct {
	Groups []Group `json:"groups"`
}

// RemoveClientAction defines the action to remove a client from a group
const RemoveClientAction = "removeClient"

// RemoveClientReq requests removal of a client from a group
// The caller must be an administrator or service.
type RemoveClientReq struct {
	ClientID  string `json:"clientID"`
	GroupName string `json:"groupName,omitempty"`
}

// RemoveClientAllAction defines the action to remove a client from all groups
const RemoveClientAllAction = "removeClientAll"

// RemoveClientAllReq requests removal of a client from all groups
// The caller must be an administrator or service.
type RemoveClientAllReq struct {
	ClientID string `json:"clientID"`
}

// SetUserRoleAction updates the role of a user in a group
const SetUserRoleAction = "setClientRole"

// SetUserRoleReq requests update of a user role in a group
// If the user is not a member of a group, the user will be added.
// The role must be one of the user roles.
type SetUserRoleReq struct {
	UserID    string `json:"userID"`
	GroupName string `json:"groupName"`
	UserRole  string `json:"userRole"`
}

// IAuthz defines the capabilities of the authorization service
type IAuthz interface {

	// AddGroup adds a new group
	// This fails if the groupName already exists
	// Use retention 0 to retain messages indefinitely
	//
	//	groupName unique name of the group
	//	retention period in seconds of events in this group
	AddGroup(groupName string, retention uint64) error

	// AddService adds a client with the service role to a group
	AddService(serviceID string, groupName string) error

	// AddThing adds a client with the thing role to a group
	AddThing(thingID string, groupName string) error

	// AddUser adds a user to a group with the user role manager, operator or viewer
	// See ClientRole...
	AddUser(userID string, role string, groupName string) error

	// DeleteGroup deletes the group and all its resources. This is not recoverable.
	DeleteGroup(groupName string) error

	// GetGroup returns the group with the given name, or an error if group is not found.
	// GroupName must not be empty and must be an existing group
	// Returns an error if the group does not exist.
	GetGroup(groupName string) (group Group, err error)

	// GetClientRoles returns a map of [groupID]role for groups the client is a member of.
	GetClientRoles(clientID string) (roles RoleMap, err error)

	// GetPermissions returns the permissions the client has for Things.
	// clientID is optional. The default is to use the connecting client's ID.
	// Only managers and services are allowed to choose a clientID different from their own.
	// Returns an map of permissions for each thing, eg PermEmitAction, etc
	GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error)

	// ListGroups returns the list of groups available to clientID
	// If clientID is "" then all groups are returned.
	ListGroups(clientID string) (groups []Group, err error)

	// RemoveClient removes a client from a group
	// The caller must be an administrator or service.
	RemoveClient(clientID string, groupName string) error

	// RemoveClientAll removes a client from all groups.
	// The caller must be an administrator or service.
	RemoveClientAll(clientID string) error

	// SetUserRole sets the role for the user in a group.
	//
	// If the client is not a member of a group the client will be added.
	// The role must be one of the user roles viewer, operator, manager
	SetUserRole(userID string, userRole string, groupName string) error
}
