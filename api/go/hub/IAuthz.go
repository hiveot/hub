package hub

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

// NewGroup creates an instance of a group with member roles
func NewGroup(groupName string) Group {
	return Group{
		Name:        groupName,
		MemberRoles: make(RoleMap),
	}
}

// Authorization management request/response messages

// AddGroupAction adds a group with a retention period
const AddGroupAction = "addGroup"

type AddGroupReq struct {
	GroupName string `json:"groupName"`
	Retention uint64 `json:"retention"`
}

// AddThingAction adds a Thing to a group
const AddThingAction = "addThing"

type AddThingReq struct {
	GroupName string `json:"groupName"`
	ThingID   string `json:"thingID"`
}

// AddServiceAction adds a service to a group
const AddServiceAction = "addService"

type AddServiceReq struct {
	GroupName string `json:"groupName"`
	ServiceID string `json:"serviceID"`
}

// AddUserAction adds a user to a group
const AddUserAction = "addUser"

type AddUserReq struct {
	GroupName string `json:"groupName"`
	UserID    string `json:"userID"`
}

// DeleteGroupAction deletes the group with the given name including its stores
const DeleteGroupAction = "deleteGroup"

type DeleteGroupReq struct {
	GroupName string `json:"groupName"`
}

// GetClientRolesAction requests a list of clients and their roles of a group
const GetClientRolesAction = "getClientRoles"

type GetClientRolesReq struct {
	GroupName string `json:"groupName"`
}
type GetClientRolesResp struct {
	Roles RoleMap `json:"roles"`
}

// GetGroupReq requests the group and its members
type GetGroupReq struct {
	GroupName string `json:"groupName"`
}

// GetGroupResp returns the group and its members
type GetGroupResp struct {
	Group Group `json:"group"`
}

// GetPermissionsReq requests the permissions a user has for a Thing
type GetPermissionsReq struct {
	ThingID []string `json:"thingID"`
}

// GetPermissionsResp returns a list of permissions the client has on a thing
// Contains an array of permissions, eg PermPubAction, etc
type GetPermissionsResp struct {
	Permissions []string `json:"permissions"`
}

// ListGroupsReq requests a list of known groups
type ListGroupsReq struct {
}

// ListGroupsResp returns a list of groups
type ListGroupsResp struct {
	Groups []Group `json:"groups"`
}

// RemoveAllReq requests removal of a client from all groups
type RemoveAllReq struct {
	ClientID string `json:"clientID"`
}

// RemoveClientReq requests removal of a client from a group
type RemoveClientReq struct {
	ClientID  string `json:"clientID"`
	GroupName string `json:"groupName"`
}
