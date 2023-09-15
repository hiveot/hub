package auth

import (
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/api/go/vocab"
)

const DefaultAclFilename = "authz.acl"

// AuthManageRolesCapability is the name of the Thing/Capability that handles role requests
const AuthManageRolesCapability = "roles"

// Predefined user roles.
const (

	// ClientRoleAdmin lets a client publish and subscribe to any sources and invoke all services
	//  Read permissions: subEvents, subActions
	//  Write permissions: pubEvents, pubActions, pubConfig
	ClientRoleAdmin = "admin"

	// ClientRoleManager lets a client subscribe to Thing TD, events, publish actions and update configuration
	//  Read permissions: subEvents
	//  Write permissions: pubActions, pubConfig
	ClientRoleManager = "manager"

	// ClientRoleNone indicates that the user has no particular role. It can not do anything until
	// the role is upgraded to viewer or better.
	//  Read permissions: none
	//  Write permissions: none
	ClientRoleNone = ""

	// ClientRoleOperator lets a client subscribe to Thing TD, events and publish actions
	//  Read permissions: subEvents
	//  Write permissions: pubActions
	ClientRoleOperator = "operator"

	// ClientRoleViewer lets a client subscribe to Thing TD and Thing Events
	//  Read permissions: subTDs, subEvents
	//  Write permissions: none
	ClientRoleViewer = "viewer"

	// ClientRoleDevice lets a client publish thing events and subscribe to device actions
	//  Read permissions: subActions
	//  Write permissions: pubTDs, pubEvents
	ClientRoleDevice = "device"
)

// viewers can subscribe to all things
var viewerPermissions = []msgserver.RolePermission{{
	Prefix:   "things",
	MsgType:  vocab.MessageTypeEvent,
	AllowPub: false,
	AllowSub: true,
}}

// devices can publish events and subscribe to actions
var devicePermissions = []msgserver.RolePermission{{
	Prefix:   "things",
	MsgType:  vocab.MessageTypeEvent,
	AllowPub: true,
	AllowSub: true,
}, {
	Prefix:   "things",
	MsgType:  vocab.MessageTypeAction,
	AllowPub: false,
	AllowSub: true,
}}

// operators can also publish thing actions
var operatorPermissions = append(viewerPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  vocab.MessageTypeAction,
	AllowPub: true,
})

// managers can also publish configuration
var managerPermissions = append(operatorPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  vocab.MessageTypeConfig,
	AllowPub: true,
})

// administrators can do all and publish to services
var adminPermissions = append(managerPermissions, msgserver.RolePermission{
	Prefix:   "svc",
	MsgType:  vocab.MessageTypeAction,
	AllowPub: true,
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]msgserver.RolePermission{
	ClientRoleViewer:   viewerPermissions,
	ClientRoleOperator: operatorPermissions,
	ClientRoleManager:  managerPermissions,
	ClientRoleAdmin:    adminPermissions,
	ClientRoleDevice:   devicePermissions,
	ClientRoleNone:     nil,
}

// capability address part used in sending messages
const AuthRolesCapability = "roles"

// CreateRoleAction defines the service action to create a new custom role
const CreateRoleAction = "createRole"

type CreateRoleReq struct {
	Role string `json:"role"`
}

// DeleteRoleAction defines the service action to delete a custom role.
const DeleteRoleAction = "deleteRole"

type DeleteRoleReq struct {
	Role string `json:"role"`
}

// SetRoleAction defines the service action to change a client's role
const SetRoleAction = "setRole"

type SetRoleReq struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

// IAuthManageRoles defines the capability to manage roles
type IAuthManageRoles interface {
	// CreateRole defines a new role with custom permissions
	// This returns an error if the role already exists
	CreateRole(roleName string) error

	// DeleteRole deletes the previously created custom role
	DeleteRole(roleName string) error

	// SetRole updates the role for the client.
	// If the role does not exist, this returns an error.
	SetRole(userID string, role string) error
}
