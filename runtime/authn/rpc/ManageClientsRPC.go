// Package rpc with RPC message structs
package rpc

import "github.com/hiveot/hub/runtime/authn"

// AddDeviceMethod is the request name to add a device with public key
const AddDeviceMethod = "addDevice"

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

// AddServiceMethod is the request name to add a service with public key
const AddServiceMethod = "addService"

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

// AddUserMethod is the request name to add a user with password
const AddUserMethod = "addUser"

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

const GetCountMethod = "getCount"

type GetCountResp struct {
	N int `json:"n"`
}

// GetClientProfileMethod is the request name to get any client's profile
const GetClientProfileMethod = "getClientProfile"

type GetClientProfileArgs struct {
	ClientID string `json:"clientID"`
}

// GetProfilesMethod is the request name to get a list of all client profiles
const GetProfilesMethod = "getProfiles"

// GetProfilesResp response to listClient actions
type GetProfilesResp struct {
	Profiles []authn.ClientProfile `json:"profiles"`
}

// RemoveClientMethod is the request name to remove a client
// The caller must be an administrator or service.
const RemoveClientMethod = "removeClient"

type RemoveClientArgs struct {
	ClientID string `json:"clientID"`
}

// SetClientPasswordMethod is the request name to set a new password for a client
// This is equivalent to UpdatePassword in Manage Profile (that the client themselves can invoke)
const SetClientPasswordMethod = "setClientPassword"

type SetClientPasswordArgs struct {
	ClientID string `json:"clientID"`
	Password string `json:"password"`
}

// UpdateClientMethod is the request name to update a client's profile
// The caller must be an administrator or service.
const UpdateClientMethod = "updateClient"

type UpdateClientArgs struct {
	ClientID string              `json:"clientID"`
	Profile  authn.ClientProfile `json:"profile"`
}

// UpdateClientRoleMethod is the request name to change a client's role
const UpdateClientRoleMethod = "updateRole"

type UpdateClientRoleArgs struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}
