package hub

import (
	"time"
)

// AuthnServiceName default ID of the service
const AuthnServiceName = "authn"

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

// AuthnConfig defines the authn service configuration
type AuthnConfig struct {

	// PasswordFile to read from. Use "" for default defined in 'unpwstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile"`

	// optional override of the default token validity periods
	DeviceTokenValiditySec  int `yaml:"deviceTokenValiditySec"`
	ServiceTokenValiditySec int `yaml:"serviceTokenValiditySec"`
	UserTokenValiditySec    int `yaml:"userTokenValiditySec"`
}

// ClientProfile contains client information
type ClientProfile struct {
	// The client ID.
	//  for users this is their email
	//  for publishers the publisherID
	//  for services the service instance ID
	ClientID string
	// ClientType identifies the client as a device, service or user
	ClientType string
	// The client presentation name
	Name string
	// IP is the IP when auth token was issued
	IP string
	// AuthTime is the ISO8601 timestamp when auth token was last issued
	AuthTime string
	// Updated is the ISO8601 timestamp the password was last updated, if any
	Updated string
}

// IManageAuthn defines the capabilities for managing authenticating clients.
// This capability is only available to administrators that connect with a valid admin
// client certificate or with an admin user token.
// Authentication is based on JWT tokens with claims for client type, validity and role.
type IManageAuthn interface {

	// AddDevice adds an IoT device and generates an authentication token
	// The device must periodically refresh its token for it to remain valid.
	//
	// This is idempotent. If the device already exists then its name is updated and a new token is returned
	//
	//  deviceID is the thingID of the device, used for publishing things by this device
	//  name of the service for presentation
	//  validity is duration the token is valid for. 0 for the default DefaultDeviceTokenValiditySec
	// This returns a new device authentication token
	//AddDevice(deviceID string, name string, validity time.Duration) (token string, err error)

	// AddService adds a service and generates a service token.
	// The service must periodically refresh its token for it to remain valid.
	//
	// This is idempotent. If the service already exists then its name is updated
	//
	//  serviceID is the instance ID of the service on the network.
	//  name of the service for presentation
	//  validity is duration the token is valid for. 0 for the default DefaultServiceTokenValiditySec
	// This returns a new service authentication token
	//AddService(serviceID string, name string, validity time.Duration) (token string, err error)

	// AddUser adds a user
	// If the userID already exists then an error is returned
	//  userID is the login ID of the user, typically their email
	//  name of the user for presentation
	//  password the user can login with if their token has expired.
	//  validity is duration the token is valid for. 0 for the default DefaultUserTokenValiditySec
	// This returns a short lived authentication token for login without requiring a password.
	AddUser(userID string, name string, password string, validity time.Duration) (err error)

	// GetProfile returns a client's profile
	GetProfile(clientID string) (profile ClientProfile, err error)

	// ListClients provide a list of known clients and their info
	ListClients() (profiles []ClientProfile, err error)

	// RemoveClient removes a client and disables authentication
	// Existing tokens are immediately expired (tbd)
	RemoveClient(clientID string) error

	// ResetPassword reset a user's login password
	ResetPassword(clientID string, password string) (err error)
}

// IClientAuthn defines the capabilities for use by authenticating clients
type IClientAuthn interface {
	// Login to obtain an auth token
	Login(clientID string, password string) (authToken string, err error)

	// Logout invalidates the authentication token and requires a login
	Logout(userID string, refreshToken string) (err error)

	// Refresh a short lived authentication token.
	//
	//  clientID is the userID, deviceID or serviceID whose token to refresh.
	//  oldToken must be a valid token obtained at login or refresh
	//
	// This returns a short lived auth token that can be used to authenticate with the hub
	// This fails if the token has expired or does not belong to the clientID
	Refresh(clientID string, oldToken string) (newToken string, err error)

	// UpdateName updates a user's name
	UpdateName(clientID string, name string) (err error)

	// UpdatePassword changes the client password
	// Login or Refresh must be called successfully first.
	UpdatePassword(clientID string, newPassword string) error

	// SetProfile updates the user profile
	// Login or Refresh must be called successfully first.
	//SetProfile(profile ClientProfile) error
}
