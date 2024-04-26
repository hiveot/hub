package authz

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
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
// agent      pub       event      {clientID}   -
//            sub       action     {clientID}   -
//            sub       properties {clientID}   -
// service    pub       -       -            -
//            sub       action  {clientID}   -
//            sub       rpc     {clientID}   -
//            sub       event   -            -

// {clientID} is replaced with the client's loginID when publishing or subscribing

// devices can publish events, replies and subscribe to their own actions and config
var agentPermissions = []api.RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AgentID:  "{clientID}", // devices can only publish their own events
		AllowPub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AgentID:  "{clientID}", // agents can only subscribe actions for themselves
		AllowSub: true,
	},
}

// viewers can subscribe to events from all things
var viewerPermissions = []api.RolePermission{{
	MsgType:  vocab.MessageTypeEvent,
	AllowSub: true,
}}

// operators can subscribe to events and publish things actions
// operators cannot configure things
var operatorPermissions = []api.RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowSub: true,
	}, {
		// action to change properties is not allowed
		MsgType:  vocab.MessageTypeAction,
		MsgKey:   vocab.ActionTypeProperties,
		AllowPub: false,
	}, {
		// any other actions are allowed
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true,
	},
}

// managers can sub all events and pub all actions
var managerPermissions = []api.RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true,
	},
}

// administrators are like managers.
// Services will add their role authorization on startup
var adminPermissions = append(managerPermissions)

// services can pub/sub anything
var servicePermissions = append(adminPermissions, api.RolePermission{
	MsgType:  vocab.MessageTypeEvent,
	AgentID:  "{clientID}",
	AllowPub: true,
}, api.RolePermission{
	MsgType:  vocab.MessageTypeAction,
	AgentID:  "{clientID}",
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]api.RolePermission{
	api.ClientRoleNone:     nil,
	api.ClientRoleAgent:    agentPermissions,
	api.ClientRoleService:  servicePermissions,
	api.ClientRoleViewer:   viewerPermissions,
	api.ClientRoleOperator: operatorPermissions,
	api.ClientRoleManager:  managerPermissions,
	api.ClientRoleAdmin:    adminPermissions,
}

// AuthzConfig holds the authorization permissions for client roles
type AuthzConfig struct {
	rolePermissions map[string][]api.RolePermission `yaml:"rolePermissions"`
	aclFile         string                          `yaml:"aclFile"`
}

// Setup ensures config is valid and loaded
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthzConfig) Setup(storesDir string) {
	if cfg.aclFile == "" {
		cfg.aclFile = api.DefaultAclFilename
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
