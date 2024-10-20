package authz

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"path"
	"sync"
)

const DefaultAclFilename = "authz.acl"

// role and their permissions. This controls whether clients can publish actions,
// publish events and subscribe to events.
//
// Specific role permissions:
// viewer:   can publish service actions that allow it
//           can not publish agent actions
//           can not publish configuration actions
//           can subscribe to thing dtw events
// operator: can publish dtw actions
//           can not publish configuration actions
//           can subscribe to thing dtw events
// manager:  can publish thing dtw actions
//           can publish thing dtw configuration actions
//           can subscribe to thing dtw events
// admin:    can publish dtw actions
//           can publish dtw configuration actions
// agent:    can publish native events (which digitwin turns into digitwin thing events)
// service:  can publish native events
//           can subscribe to any dtw events
//
// Services set their own default permissions on what roles can use them. Some examples:
// digitwin directory service:
//            readTD, readTDs, QueryDTDs methods: all roles
//            removeTD: manager, admin, service
// digitwin inbox: (action store)
//            readLatest: operator, manager, admin, service
// digitwin outbox: (event store)
// 	          readLatest: all roles
//            removeValue: manager, admin, service
// state storage:
//			  all roles can use this. Service limits it to the client's own data.
// history read:
//            all roles can read history
// history manage:
//            manager, admin, service

// RolePermission defines authorization for a role.
// Each permission defines the source/things the user can pub/sub to.
type RolePermission struct {
	//// device or service publishing the Thing data, or "" for all
	//AgentID string
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

// agents can publish actions, events, property updates, and subscribe to their own actions and config
var agentPermissions = []RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowPub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true, // only allow pub to select services
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeProperty,
		AllowPub: true, // publish property updates
		AllowSub: true, // receive property write requests
	},
}

// services can pub/sub anything
var servicePermissions = []RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowPub: true,
		AllowSub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AllowSub: true,
		AllowPub: true,
	}, {
		MsgType:  vocab.MessageTypeProperty,
		AllowSub: true,
		AllowPub: true,
	},
}

// viewers can subscribe to events from all things
var viewerPermissions = []RolePermission{{
	MsgType:  vocab.MessageTypeEvent,
	AllowSub: true,
}}

// operators can subscribe to events and publish things actions
// operators cannot configure things
var operatorPermissions = []RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowSub: true,
	}, {
		// action to change properties is not allowed
		MsgType:  vocab.MessageTypeProperty,
		AllowPub: true,
	}, {
		// any other actions are allowed
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true,
	},
}

// managers can sub all events and pub all actions
var managerPermissions = []RolePermission{
	{
		MsgType:  vocab.MessageTypeEvent,
		AllowSub: true,
	}, {
		// action to change properties is not allowed
		MsgType:  vocab.MessageTypeProperty,
		AllowPub: true,
	}, {
		MsgType:  vocab.MessageTypeAction,
		AllowPub: true,
	},
}

// administrators are like managers.
// Services will add their role authorization on startup
var adminPermissions = append(managerPermissions)

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[authz.ClientRole][]RolePermission{
	authz.ClientRoleNone:     nil,
	authz.ClientRoleAgent:    agentPermissions,
	authz.ClientRoleService:  servicePermissions,
	authz.ClientRoleViewer:   viewerPermissions,
	authz.ClientRoleOperator: operatorPermissions,
	authz.ClientRoleManager:  managerPermissions,
	authz.ClientRoleAdmin:    adminPermissions,
}

// AuthzConfig holds the authorization permissions for client roles
type AuthzConfig struct {
	// map of role to permissions of that role
	RolePermissions map[authz.ClientRole][]RolePermission `yaml:"RolePermissions"`

	// map of service dThingID  to the allow/deny roles that can invoke it
	ThingPermissions map[string]authz.ThingPermissions `yaml:"ServicePermissions"`

	// file with configured permissions
	aclFile string `yaml:"aclFile"`

	// mutex for accessing configuration
	mux sync.RWMutex
}

func (cfg *AuthzConfig) GetPermissions(dThingID string) (authz.ThingPermissions, bool) {
	cfg.mux.Lock()
	defer cfg.mux.Unlock()
	perm, found := cfg.ThingPermissions[dThingID]
	return perm, found
}

// SetPermissions defines Thing specific permissions
// Intended for storing digital-twin thing permissions
func (cfg *AuthzConfig) SetPermissions(perms authz.ThingPermissions) {
	cfg.mux.Lock()
	defer cfg.mux.Unlock()
	dThingID := tdd.MakeDigiTwinThingID(perms.AgentID, perms.ThingID)
	cfg.ThingPermissions[dThingID] = perms
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
		RolePermissions:  DefaultRolePermissions,
		ThingPermissions: make(map[string]authz.ThingPermissions),
	}
	return cfg
}
