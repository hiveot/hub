// Package authz with authorization definitions
package authz

const DefaultAclFilename = "authz.acl"

// AuthManageRolesCapability is the name of the Thing/Capability that handles role requests
const AuthManageRolesCapability = "manageRoles"

// Predefined user roles.
const (

	// ClientRoleNone indicates that the user has no particular role. It can not do anything until
	// the role is upgraded to viewer or better.
	//  Read permissions: none
	//  Write permissions: none
	ClientRoleNone = ""

	// ClientRoleAdmin lets a client publish and subscribe to any sources and invoke all services
	//  Read permissions: subEvents, subActions
	//  Write permissions: pubEvents, pubActions, pubConfig
	ClientRoleAdmin = "admin"

	// ClientRoleAgent lets a device agent publish thing events and subscribe to device actions
	//  Read permissions: subActions
	//  Write permissions: pubTDs, pubEvents
	ClientRoleAgent = "agent"

	// ClientRoleManager lets a client subscribe to Thing TD, events, publish actions and update configuration
	//  Read permissions: subEvents
	//  Write permissions: pubActions, pubConfig
	ClientRoleManager = "manager"

	// ClientRoleOperator lets a client subscribe to events and publish actions
	//  Read permissions: subEvents
	//  Write permissions: pubActions
	ClientRoleOperator = "operator"

	// ClientRoleService lets a client acts as an admin user and a device
	//  Read permissions: subEvents, subActions, subConfig
	//  Write permissions: pubEvents, pubActions, pubConfig
	ClientRoleService = "service"

	// ClientRoleViewer lets a client subscribe to Thing TD and Thing Events
	//  Read permissions: subEvents
	//  Write permissions: none
	ClientRoleViewer = "viewer"
)

// RolePermission defines authorization for a role.
// Each permission defines the source/things the user can pub/sub to.
type RolePermission struct {
	AgentID  string // device or service publishing the Thing data, or "" for all
	ThingID  string // thingID or capability, or "" for all
	MsgType  string // rpc, event, action, config, or "" for all message types
	MsgName  string // action name or "" for all actions
	AllowPub bool   // allow publishing of this message
	AllowSub bool   // allow subscribing to this message
}

// AuthRolesCapability defines the 'capability' address part used in sending messages
const AuthRolesCapability = "roles"

// CreateRoleReq defines the request to create a new custom role
const CreateRoleReq = "createRole"

type CreateRoleArgs struct {
	Role string `json:"role"`
}

// DeleteRoleReq defines the request to delete a custom role.
const DeleteRoleReq = "deleteRole"

type DeleteRoleArgs struct {
	Role string `json:"role"`
}
