package service

import (
	"path"
	"sync"

	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	authz "github.com/hiveot/hub/runtime/authz/api"
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
	// device or service publishing the Thing data, or "" for all
	//AgentID string
	// thingID or capability, or "" for all
	ThingID string `yaml:"thingID"`
	// OpSubscribeEvent,... action, config, or "" for all message types
	Operations []string `yaml:"operations"`
	// action name or "" for all actions
	//MsgKey string
}

// re-usable auth permissions to refresh token and logout
var authPermissions = []string{
	//wot.HTOpLogout,
	//wot.HTOpRefresh,
}

// re-usable permissions to observe and read and observe properties, events, actions, TDs
var readPermissions = []string{
	//wot.HTOpReadEvent, wot.HTOpReadAllEvents,
	//wot.HTOpReadTD, wot.HTOpReadAllTDs,
	wot.OpObserveProperty, wot.OpObserveAllProperties,
	//wot.OpQueryAction, wot.OpQueryAllActions,  // why query actions if you cant invoke?
	wot.OpReadProperty, wot.OpReadAllProperties,
	wot.OpSubscribeEvent, wot.OpSubscribeAllEvents,
	wot.OpUnobserveProperty, wot.OpUnobserveAllProperties,
	wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents,
}

// Thing agents can refresh tokens and invoke actions on services.
var agentPermissions = RolePermission{
	Operations: append(authPermissions,
		wot.OpInvokeAction,
	),
}

// services can do almost anything
var servicePermissions = RolePermission{
	Operations: append(
		append(authPermissions, readPermissions...),
		wot.OpQueryAction, wot.OpQueryAllActions,
		wot.OpInvokeAction, wot.OpWriteProperty,
	),
}

// viewers can authenticate and read properties and events
var viewerPermissions = RolePermission{
	Operations: append(authPermissions, readPermissions...),
}

// operators can subscribe to events and publish things actions
// operators cannot configure things
var operatorPermissions = RolePermission{
	Operations: append(
		append(authPermissions, readPermissions...),
		vocab.OpInvokeAction,
		wot.OpQueryAction, wot.OpQueryAllActions,
	),
}

// managers are operators that can also configure properties
var managerPermissions = RolePermission{
	Operations: append(operatorPermissions.Operations,
		vocab.OpWriteProperty,
	),
}

// administrators are like managers.
// Services will add their role authorization on startup
var adminPermissions = RolePermission{
	Operations: append(managerPermissions.Operations), // copy the permissions
}

// DefaultRolePermissions contains the default pub/sub permissions for each role
var DefaultRolePermissions = map[authz.ClientRole]RolePermission{
	authz.ClientRoleNone:     {},
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
	RolePermissions map[authz.ClientRole]RolePermission `yaml:"rolePermissions"`

	// map of service dThingID  to the allow/deny roles that can invoke it
	ThingPermissions map[string]authz.ThingPermissions `yaml:"servicePermissions"`

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
	dThingID := td.MakeDigiTwinThingID(perms.AgentID, perms.ThingID)
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
