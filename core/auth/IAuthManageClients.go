package auth

// AuthServiceName default ID of the authentication and authorization service
const AuthServiceName = "auth"

// AuthManageClientsCapability is the name of the Thing/Capability that handles management requests
const AuthManageClientsCapability = "clients"

// Types of clients that are issued authentication tokens
const (
	ClientTypeDevice  = "device"
	ClientTypeService = "service"
	ClientTypeUser    = "user"
)

// Authentication token validity for client types
const (
	DefaultDeviceTokenValidityDays  = 90  // 90 days
	DefaultServiceTokenValidityDays = 365 // 1 year
	DefaultUserTokenValidityDays    = 30  // 30 days
)

// DefaultAdminUserID is the client ID of the default administrator account
const DefaultAdminUserID = "admin"

// DefaultLauncherServiceID is the client ID of the launcher service
// auth creates a key and auth token for the launcher on startup
const DefaultLauncherServiceID = "launcher"

// ClientProfile contains client information of sources and users
type ClientProfile struct {
	// The client ID.
	//  for users this is their email
	//  for IoT devices or services, use the bindingID
	//  for services the service instance ID
	ClientID string
	// ClientType identifies the client as a ClientTypeDevice, ClientTypeService or ClientTypeUser
	ClientType string
	// The client presentation name
	DisplayName string
	// The client's public key
	PubKey string
	// timestamp in ISO8601 format, the entry was last updated
	Updated string
	// TokenValidityDays nr of days that issued JWT tokens are valid for or 0 for default
	TokenValidityDays int
	// The client's role
	Role string
}

// Authentication management request/response messages

// AddDeviceReq defines the request to add a device with public key
const AddDeviceReq = "addDevice"

// AddDeviceArgs request message to add a device.
// The caller must be an administrator or service.
type AddDeviceArgs struct {
	DeviceID    string `json:"deviceID"`
	DisplayName string `json:"displayName"`
	PubKey      string `json:"pubKey"`
}
type AddDeviceResp struct {
	Token string `json:"token"`
}

// AddServiceReq defines the request to add a service with public key
const AddServiceReq = "addService"

// AddServiceArgs request message to add a service.
// The caller must be an administrator or service.
type AddServiceArgs struct {
	ServiceID   string `json:"serviceID"`
	DisplayName string `json:"displayName"`
	PubKey      string `json:"pubKey"`
}
type AddServiceResp struct {
	Token string `json:"token"`
}

// AddUserReq defines the service request to add a user with password
const AddUserReq = "addUser"

// AddUserArgs request message to add a user.
// The caller must be an administrator or service.
type AddUserArgs struct {
	UserID      string `json:"userID"`
	DisplayName string `json:"DisplayName,omitempty"`
	Password    string `json:"password,omitempty"`
	PubKey      string `json:"pubKey,omitempty"`
	Role        string `json:"role,omitempty"`
}
type AddUserResp struct {
	Token string `json:"token"`
}

const GetCountReq = "getCount"

type GetCountResp struct {
	N int `json:"n"`
}

// GetClientProfileReq defines the request to get any client's profile
const GetClientProfileReq = "getClientProfile"

type GetClientProfileArgs struct {
	ClientID string `json:"clientID"`
}

// GetProfilesReq defines the service action to get a list of all client profiles
const GetProfilesReq = "getProfiles"

// GetProfilesResp response to listClient actions
type GetProfilesResp struct {
	Profiles []ClientProfile `json:"profiles"`
}

// RemoveClientReq defines the request to remove a client
// The caller must be an administrator or service.
const RemoveClientReq = "removeClient"

type RemoveClientArgs struct {
	ClientID string `json:"clientID"`
}

// UpdateClientReq defines the request to update a client's profile
// The caller must be an administrator or service.
const UpdateClientReq = "updateClient"

type UpdateClientArgs struct {
	ClientID string        `json:"clientID"`
	Profile  ClientProfile `json:"profile"`
}

// UpdateClientPasswordReq defines the service action to update a client's password
// The caller must be an administrator or service.
const UpdateClientPasswordReq = "updateClientPassword"

type UpdateClientPasswordArgs struct {
	ClientID string `json:"clientID"`
	Password string `json:"password"`
}

// UpdateClientRoleReq defines the service action to change a client's role
const UpdateClientRoleReq = "updateRole"

type UpdateClientRoleArgs struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

// IAuthnManageClients defines the capabilities for managing authenticating clients.
// This capability is only available to administrators.
// Authentication is based on JWT tokens with claims for client type, validity and role.
type IAuthnManageClients interface {

	// AddDevice adds an IoT device and generates an authentication token
	// The device must periodically refresh its token for it to remain valid.
	// This returns a new device authentication token
	//
	// The format of the token depends on the server configuration. NKey servers return
	// the public key, jwt servers return a jwt based token.
	//
	// If the device already exists or the pubKey is invalid then an error is returned.
	//
	//  deviceID is the thingID of the device, used for publishing things by this device.
	//  displayName of the service for presentation
	//  pubKey ECDSA public key of the device
	AddDevice(deviceID string, displayName string, pubKey string) (token string, err error)

	// AddService adds a new service and generates a service token.
	// The service must periodically refresh its token for it to remain valid.
	// This returns a new service authentication token.
	//
	// The format of the token depends on the server configuration. NKey servers return
	// the public key, jwt servers return a jwt based token.
	//
	// If the serviceID already exists or the public key is invalid then an error is returned
	//
	//  serviceID is the instance ID of the service on the network.
	//  displayName of the service for presentation
	//  pubKey ECDSA public key of the service
	AddService(serviceID string, displayName string, pubKey string) (token string, err error)

	// AddUser adds a user with a password, public key or neither.
	// The caller must be an administrator or service.
	// If the userID already exists the the user is updated.
	// This returns a new user authentication token if a valid pubKey is provided.
	//
	// The format of the token depends on the server configuration. NKey servers return
	// the public key, jwt servers return a jwt based token.
	//
	//  userID is the login ID of the user, typically their email
	//  displayName of the user for presentation
	//  password the user can login with if their token has expired. Optional.
	//  pubKey the public key to receive a signed token. Ignored if not a valid key.
	AddUser(userID string, displayName string, password string, pubKey string, role string) (token string, err error)

	// GetAuthClientList provides a list of clients to apply to the message server
	//GetAuthClientList() []msgserver.AuthClient

	// GetCount returns the number of users in the store
	GetCount() (int, error)

	// GetProfile returns a user's profile
	// This returns an error if the user does not exist
	GetProfile(clientID string) (profile ClientProfile, err error)

	// GetProfiles provide a list of known clients and their info.
	// The caller must be an administrator or service.
	GetProfiles() (profiles []ClientProfile, err error)

	// RemoveClient removes a user and disables authentication
	// Existing tokens are immediately expired (tbd)
	RemoveClient(clientID string) error

	// UpdateClient updates a client's profile
	UpdateClient(clientID string, prof ClientProfile) error

	// UpdateClientPassword updates the password for the client.
	UpdateClientPassword(clientID string, newPass string) error

	// UpdateClientRole updates the role for the client.
	// If the role does not exist, this returns an error.
	UpdateClientRole(clientID string, role string) error
}