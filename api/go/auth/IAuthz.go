package auth

const DefaultAclFilename = "authz.acl"

// AuthzCapability is the name of the Thing/Capability that handles authorization requests
const AuthzCapability = "authz"

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
)

// IAuthz defines the capability to authorize users
type IAuthz interface {
	// RegisterRole defines a new role with custom permissions
	RegisterRole(roleName string)

	// RegisterService authorizes the use of the service to select roles.
	// This is invoked by a service on startup to ensure the right users can access it.
	// capRoles is a map of the service capability to the roles that can use it.
	//
	//  sourceID is the device/service ID of the source
	//  capRoles is a map of capabilityID:roles that can use the service capability
	RegisterService(sourceID string, capRoles map[string][]string)

	// SetRole updates the role for the user.
	// If the role does not exist, this returns an error.
	SetRole(userID string, role string) error
}
