package auth

import (
	"time"
)

// AuthnServiceName default ID of the authentication and authorization service
const AuthServiceName = "auth"

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
	DefaultDeviceTokenValidity  = time.Hour * 24 * 90  // 90 days
	DefaultServiceTokenValidity = time.Hour * 24 * 365 // 1 year
	DefaultUserTokenValidity    = time.Hour * 24 * 30  // 30 days
)

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
	// TokenValidity time that issued JWT tokens are valid for or 0 for default
	TokenValidity time.Duration
	// The client's role
	Role string
}

// Authentication management request/response messages

// AddDeviceAction defines the action to add a device with public key
const AddDeviceAction = "addDevice"

// AddDeviceReq request message to add a device.
// The caller must be an administrator or service.
type AddDeviceReq struct {
	DeviceID      string        `json:"deviceID"`
	DisplayName   string        `json:"displayName"`
	PubKey        string        `json:"pubKey"`
	TokenValidity time.Duration `json:"tokenValidity"`
}
type AddDeviceResp struct {
	Token string `json:"token"`
}

// AddServiceAction defines the action to add a service with public key
const AddServiceAction = "addService"

// AddServiceReq request message to add a service.
// The caller must be an administrator or service.
type AddServiceReq struct {
	ServiceID     string        `json:"serviceID"`
	DisplayName   string        `json:"displayName"`
	PubKey        string        `json:"pubKey"`
	TokenValidity time.Duration `json:"tokenValidity"`
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
// This capability is only available to administrators.
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
	//  tokenValidity is duration the device token is valid for. 0 for the default DefaultDeviceTokenValiditySec
	AddDevice(deviceID string, displayName string, pubKey string, tokenValidity time.Duration) (token string, err error)

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
	//  tokenValidity is duration the service token is valid for. 0 for the default DefaultServiceTokenValiditySec
	AddService(serviceID string, displayName string, pubKey string, tokenValidity time.Duration) (token string, err error)

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
	AddUser(userID string, displayName string, password string, pubKey string) (token string, err error)

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

	// RemoveUser removes a user and disables authentication
	// Existing tokens are immediately expired (tbd)
	RemoveUser(userID string) error

	// UpdateUser updates a user's profile
	UpdateUser(userID string, prof ClientProfile) error
}
