package api

//const AuthzAgentID = "authz"
//
//// AuthzAdminServiceID is the ThingID of service to administer authorization
//const AuthzAdminServiceID = "admin"
//
//// AuthzUserServiceID is the ThingID of the user facing service
//const AuthzUserServiceID = "user"

// internal
const DefaultAclFilename = "authz.acl"

//// supported methods for authz management service
//const (
//	GetClientRoleMethod = "getClientRole"
//	SetClientRoleMethod = "setClientRole"
//)
//
//// supported methods for authz user services
//const (
//	// SetPermissionsMethod set the permissions for roles to use a service/thing
//	// Intended for use by services.
//	// This sets the user roles that are allowed to use the service.
//	// This fails if the client is not a service.
//	SetPermissionsMethod = "setPermissions"
//)

// Predefined user roles.
//const (
//
//	// ClientRoleNone indicates that the user has no particular role. It can not do anything until
//	// the role is upgraded to viewer or better.
//	//  Read permissions: none
//	//  Write permissions: none
//	ClientRoleNone = ""
//
//	// ClientRoleAdmin lets a client publish and subscribe to any sources and invoke all services
//	//  Read permissions: subEvents, subActions
//	//  Write permissions: pubEvents, pubActions, pubConfig
//	ClientRoleAdmin = "admin"
//
//	// ClientRoleAgent lets a device agent publish thing events and subscribe to device actions
//	//  Read permissions: subActions
//	//  Write permissions: pubTDs, pubEvents
//	ClientRoleAgent = "agent"
//
//	// ClientRoleManager lets a client subscribe to Thing TD, events, publish actions and update configuration
//	//  Read permissions: subEvents
//	//  Write permissions: pubActions, pubConfig
//	ClientRoleManager = "manager"
//
//	// ClientRoleOperator lets a client subscribe to events and publish actions
//	//  Read permissions: subEvents
//	//  Write permissions: pubActions
//	ClientRoleOperator = "operator"
//
//	// ClientRoleService lets a client acts as an admin user and a device
//	//  Read permissions: subEvents, subActions, subConfig
//	//  Write permissions: pubEvents, pubActions, pubConfig
//	ClientRoleService = "service"
//
//	// ClientRoleViewer lets a client subscribe to Thing TD and Thing Events
//	//  Read permissions: subEvents
//	//  Write permissions: none
//	ClientRoleViewer = "viewer"
//)

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

//
//type GetClientRoleArgs struct {
//	ClientID string `json:"clientID"`
//}
//
//type GetClientRoleResp struct {
//	ClientID string `json:"clientID"`
//	Role     string `json:"role"`
//}
//
//type SetClientRoleArgs struct {
//	ClientID string `json:"clientID"`
//	Role     string `json:"role"`
//}
//
//// Createa custom role
//type CreateRoleArgs struct {
//	Role string `json:"role"`
//}
//
//type DeleteRoleArgs struct {
//	Role string `json:"role"`
//}

// ThingPermissions contains the arguments for authorizing the use of a Thing.
//
// Used by agents to set the roles that can invoke actions on a service.
// These permissions are default recommendations made by the service provider. The
// authz service can override these defaults with another configuration.
//
// With no permissions set, the result of role permissions applies.
// When a service permission is set, the default role permissions do not apply.
//type ThingPermissions struct {
//	// ThingID is the ThingID of the service as defined by its agent
//	ThingID string `json:"thingID"`
//
//	// Allow maps service keys to roles allowed to invoke the action
//	// The empty key "" applies to all actions.
//	Allow []string `json:"allow"`
//	// Allow maps service keys to roles denied to invoke the action
//	Deny []string `json:"deny"`
//}
