package authn

// AuthnServiceName default ID of the service (bindingID)
const AuthnServiceName = "authn"

// ManageAuthnCapability is the name of the Thing/Capability that handles management requests
const ManageAuthnCapability = "manage"

// Types of clients that are issued authentication tokens
const (
	ClientTypeDevice  = "device"
	ClientTypeService = "service"
	ClientTypeUser    = "user"
)

// Authentication token validity for client types
const (
	DefaultDeviceTokenValiditySec  = 90 * 24 * 3600  // 90 days
	DefaultServiceTokenValiditySec = 365 * 24 * 3600 // 1 year
	DefaultUserTokenValiditySec    = 30 * 24 * 3600  // 30 days
)

// ClientProfile contains client information
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
	// ValiditySec time that issued authn tokens are valid for or 0 for default
	ValiditySec int
}

// Authentication management request/response messages

// AddDeviceAction defines the action to add a device with public key
const AddDeviceAction = "addDevice"

// AddDeviceReq request message to add a device.
// The caller must be an administrator or service.
type AddDeviceReq struct {
	DeviceID    string `json:"deviceID"`
	DisplayName string `json:"displayName"`
	PubKey      string `json:"pubKey"`
	ValiditySec int    `json:"validitySec"`
}
type AddDeviceResp struct {
	Token string `json:"token"`
}

// AddServiceAction defines the action to add a service with public key
const AddServiceAction = "addService"

// AddServiceReq request message to add a service.
// The caller must be an administrator or service.
type AddServiceReq struct {
	ServiceID   string `json:"serviceID"`
	DisplayName string `json:"displayName"`
	PubKey      string `json:"pubKey"`
	ValiditySec int    `json:"validitySec"`
}
type AddServiceResp struct {
	Token string `json:"token"`
}

// AddUserAction defines the action to add a user with password
const AddUserAction = "addUser"

// AddUserReq request message to add a user.
// The caller must be an administrator or service.
type AddUserReq struct {
	UserID      string `json:"userID"`
	DisplayName string `json:"DisplayName,omitempty"`
	Password    string `json:"password,omitempty"`
	PubKey      string `json:"pubKey,omitempty"`
}
type AddUserResp struct {
	Token string `json:"token"`
}

const GetCountAction = "getCount"

type GetCountResp struct {
	N int `json:"n"`
}

// GetProfilesAction defines the action to get a list of all client profiles
const GetProfilesAction = "getProfiles"

// GetProfilesResp response to listClient actions
type GetProfilesResp struct {
	Profiles []ClientProfile `json:"profiles"`
}

// RemoveClientAction defines the action to remove a client
// The caller must be an administrator or service.
const RemoveClientAction = "removeClient"

type RemoveClientReq struct {
	ClientID string `json:"clientID"`
}

// UpdateClientAction defines the action to update a client's profile
// The caller must be an administrator or service.
const UpdateClientAction = "updateClient"

type UpdateClientReq struct {
	ClientID string        `json:"clientID"`
	Profile  ClientProfile `json:"profile"`
}

// IAuthnManage defines the capabilities for managing authenticating clients.
// This capability is only available to administrators that connect with a valid admin
// client certificate or with an admin user token.
// Authentication is based on JWT tokens with claims for client type, validity and role.
type IAuthnManage interface {

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
	//  validitySec is duration the device token is valid for. 0 for the default DefaultDeviceTokenValiditySec
	AddDevice(deviceID string, displayName string, pubKey string, validitySec int) (token string, err error)

	// AddService adds a new service and generates a service token.
	// The service must periodically refresh its token for it to remain valid.
	// This returns a new service authentication token
	//
	// The format of the token depends on the server configuration. NKey servers return
	// the public key, jwt servers return a jwt based token.
	//
	// If the serviceID already exists or the public key is invalid then an error is returned
	//
	//  serviceID is the instance ID of the service on the network.
	//  displayName of the service for presentation
	//  pubKey ECDSA public key of the service
	//  validitySec is duration the service token is valid for. 0 for the default DefaultServiceTokenValiditySec
	AddService(serviceID string, displayName string, pubKey string, validitySec int) (token string, err error)

	// AddUser adds a user with a password, public key or neither.
	// The caller must be an administrator or service.
	// If the userID already exists or the pubKye is invalid then an error is returned
	// This returns a new user authentication token if a valid pubKey is provided.
	//
	// The format of the token depends on the server configuration. NKey servers return
	// the public key, jwt servers return a jwt based token.
	//
	//  userID is the login ID of the user, typically their email
	//  displayName of the user for presentation
	//  password the user can login with if their token has expired. Optional.
	//  pubKey the public key to receive a signed token. Ignored if not a valid key.
	//  validitySec is duration the token is valid for. 0 for the default DefaultServiceTokenValiditySec
	AddUser(userID string, displayName string, password string, pubKey string) (token string, err error)

	// GetCount returns the number of clients in the store
	GetCount() (int, error)

	// GetProfile returns a client's profile
	// Users can only get their own profile.
	// Managers can get other clients profiles.
	// This returns an error if the client does not exist
	GetProfile(clientID string) (profile ClientProfile, err error)

	// GetProfiles provide a list of known clients and their info.
	// The caller must be an administrator or service.
	GetProfiles() (profiles []ClientProfile, err error)

	// RemoveClient removes a client and disables authentication
	// Existing tokens are immediately expired (tbd)
	RemoveClient(clientID string) error

	// UpdateClient updates a client's profile
	UpdateClient(clientID string, prof ClientProfile) error
}
