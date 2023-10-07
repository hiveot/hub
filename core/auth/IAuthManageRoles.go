package auth

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/msgserver"
)

const DefaultAclFilename = "authz.acl"

// AuthManageRolesCapability is the name of the Thing/Capability that handles role requests
const AuthManageRolesCapability = "roles"

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

	// ClientRoleDevice lets a client publish thing events and subscribe to device actions
	//  Read permissions: subActions
	//  Write permissions: pubTDs, pubEvents
	ClientRoleDevice = "device"

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

// Role based ACL matrix example
// -----------------------------
// role       pub/sub   stype   deviceID    thingID
//
// *	      sub       _INBOX  {clientID}   -       	(built-in rule)
// *	      pub       rpc     auth         profile 	(built-in rule)
// *          pub       any     -            -        senderID must be clientID except for inbox
//
// viewer     sub       event   -            -
// operator   pub       action  -            -
//            sub       event   -            -
// manager    pub       action  -            -
//            pub       config  -            -
//            sub       event   -            -
// admin      pub       action  -            -
//            sub       event   -            -
// device     pub       event   {clientID}   -
//            sub       event   -		     -
//            sub       action  {clientID}   -
// service    pub       -       -            -
//            sub       action  {clientID}   -
//            sub       rpc     {clientID}   -
//            sub       event   -            -

// {clientID} is replaced with the client's loginID when publishing or subscribing

// devices can publish events, replies and subscribe to their own actions and config
var devicePermissions = []msgserver.RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		DeviceID: "{clientID}", // devices can only publish their own events
		AllowPub: true,
	}, {
		MsgType:  vocab.MessageTypeEvent,
		DeviceID: "", // devices can subscribe to events
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		DeviceID: "{clientID}",
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeConfig,
		DeviceID: "{clientID}",
		AllowSub: true,
	},
}

// viewers can subscribe to all things
var viewerPermissions = []msgserver.RolePermission{{
	MsgType:  vocab.MessageTypeEvent,
	AllowSub: true,
}}

// operators can subscribe to events and publish thing actions
var operatorPermissions = []msgserver.RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true,
	},
}

// managers can in addition to operator also publish configuration
var managerPermissions = append(operatorPermissions, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeConfig,
	AllowPub: true,
})

// administrators can in addition to operators publish all RPCs
// RPC request permissions for roles are set by the service when they register.
var adminPermissions = append(managerPermissions, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeRPC,
	AllowPub: true,
})

// services are admins that can also publish events and subscribe to their own rpc, actions and config
var servicePermissions = append(adminPermissions, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeEvent,
	DeviceID: "{clientID}",
	AllowPub: true,
}, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeRPC,
	DeviceID: "{clientID}",
	AllowSub: true,
}, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeAction,
	DeviceID: "{clientID}",
	AllowSub: true,
}, msgserver.RolePermission{
	MsgType:  vocab.MessageTypeConfig,
	DeviceID: "{clientID}",
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]msgserver.RolePermission{
	ClientRoleNone:     nil,
	ClientRoleDevice:   devicePermissions,
	ClientRoleService:  servicePermissions,
	ClientRoleViewer:   viewerPermissions,
	ClientRoleOperator: operatorPermissions,
	ClientRoleManager:  managerPermissions,
	ClientRoleAdmin:    adminPermissions,
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

// IAuthManageRoles defines the capability to manage custom roles
type IAuthManageRoles interface {
	// CreateRole defines a new role with custom permissions
	// This returns an error if the role already exists
	CreateRole(roleName string) error

	// DeleteRole deletes the previously created custom role
	DeleteRole(roleName string) error
}
