package auth

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
)

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
