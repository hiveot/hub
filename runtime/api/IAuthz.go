// Package authz with authorization definitions
package api

const AuthzThingID = "authz"

const DefaultAclFilename = "authz.acl"

// supported methods for authz service
const (
	GetClientRoleMethod = "getClientRole"
	SetClientRoleMethod = "setClientRole"
)

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
	// device or service publishing the Thing data, or "" for all
	AgentID string
	// thingID or capability, or "" for all
	ThingID string
	// rpc, event, action, config, or "" for all message types
	MsgType string
	// action name or "" for all actions
	MsgKey string
	// allow publishing of this message
	AllowPub bool
	// allow subscribing to this message
	AllowSub bool
}

type GetClientRoleArgs struct {
	ClientID string `json:"clientID"`
}

type GetClientRoleResp struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

type SetClientRoleArgs struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

type CreateRoleArgs struct {
	Role string `json:"role"`
}

type DeleteRoleArgs struct {
	Role string `json:"role"`
}
