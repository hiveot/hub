package authz

import (
	"github.com/hiveot/hub/lib/hubclient/transports"
	"path"
)

// Role based ACL matrix example
// -----------------------------
// role       pub/sub   stype   agentID    thingID
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
// agent      pub       event   {clientID}   -
//            sub       action  {clientID}   -
// service    pub       -       -            -
//            sub       action  {clientID}   -
//            sub       rpc     {clientID}   -
//            sub       event   -            -

// {clientID} is replaced with the client's loginID when publishing or subscribing

// devices can publish events, replies and subscribe to their own actions and config
var agentPermissions = []RolePermission{
	{
		MsgType:  transports.MessageTypeEvent,
		AgentID:  "{clientID}", // devices can only publish their own events
		AllowPub: true,
	}, {
		MsgType:  transports.MessageTypeAction,
		AgentID:  "{clientID}", // agents can only subscribe actions for themselves
		AllowSub: true,
	}, {
		MsgType:  transports.MessageTypeConfig,
		AgentID:  "{clientID}",
		AllowSub: true,
	},
}

// viewers can subscribe to all things
var viewerPermissions = []RolePermission{{
	MsgType:  transports.MessageTypeEvent,
	AllowSub: true,
}}

// operators can subscribe to events and publish things actions
var operatorPermissions = []RolePermission{
	{
		MsgType:  transports.MessageTypeEvent,
		AllowSub: true,
	}, {
		MsgType:  transports.MessageTypeAction,
		AllowPub: true,
	},
}

// managers can in addition to operator also publish configuration
var managerPermissions = append(operatorPermissions, RolePermission{
	MsgType:  transports.MessageTypeConfig,
	AllowPub: true,
})

// administrators can in addition to operators publish all RPCs
// RPC request permissions for roles are set by the service when they register.
var adminPermissions = append(managerPermissions, RolePermission{
	MsgType:  transports.MessageTypeRPC,
	AllowPub: true,
})

// services are admins that can also publish events and subscribe to their own rpc, actions and config
var servicePermissions = append(adminPermissions, RolePermission{
	MsgType:  transports.MessageTypeEvent,
	AgentID:  "{clientID}",
	AllowPub: true,
}, RolePermission{
	MsgType:  transports.MessageTypeRPC,
	AgentID:  "{clientID}",
	AllowSub: true,
}, RolePermission{
	MsgType:  transports.MessageTypeAction,
	AgentID:  "{clientID}",
	AllowSub: true,
}, RolePermission{
	MsgType:  transports.MessageTypeConfig,
	AgentID:  "{clientID}",
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]RolePermission{
	ClientRoleNone:     nil,
	ClientRoleAgent:    agentPermissions,
	ClientRoleService:  servicePermissions,
	ClientRoleViewer:   viewerPermissions,
	ClientRoleOperator: operatorPermissions,
	ClientRoleManager:  managerPermissions,
	ClientRoleAdmin:    adminPermissions,
}

// AuthzConfig holds the authorization permissions for client roles
type AuthzConfig struct {
	rolePermissions map[string][]RolePermission `yaml:"rolePermissions"`
	aclFile         string                      `yaml:"aclFile"`
}

// Setup ensures config is valid and loaded
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthzConfig) Setup(storesDir string) {
	if cfg.aclFile == "" {
		cfg.aclFile = DefaultAclFilename
	}
	if !path.IsAbs(cfg.aclFile) {
		cfg.aclFile = path.Join(storesDir, "authz", cfg.aclFile)
	}
}

func NewAuthzConfig() AuthzConfig {
	cfg := AuthzConfig{
		rolePermissions: DefaultRolePermissions,
	}
	return cfg
}
