// Package rpc with RPC message structs
package rpc

import (
	"github.com/hiveot/hub/runtime/api"
)

// AuthProfileCapability is the name of the Thing/Capability that handles client requests
const AuthProfileCapability = "profile"

// below a list of actions and their payload

// GetProfileMethod defines the request to get the current client's profile
const GetProfileMethod = "getProfile"

// GetProfileResp response message to get the client's profile.
// The message address MUST contain the client sending the action to whom this applies
type GetProfileResp struct {
	Profile api.ClientProfile `json:"profile"`
}

// NewTokenMethod requests a new jwt token for password based login
// This returns a short-lived auth token that can be used to connect to the message server
// The token can be refreshed to extend it without requiring a login password.
// A public key must be on file for this to work.
const NewTokenMethod = "newToken"

type NewTokenArgs struct {
	// Password to verify identity
	Password string `json:"password"`
}
type NewTokenResp struct {
	Token string `json:"Token"`
}

// RefreshTokenMethod requests a new token for the current client
//
// This returns a new short-lived auth token that can be used to authenticate with the hub
// This requires the client's public key on file.
const RefreshTokenMethod = "refresh"

type RefreshTokenResp struct {
	Token string `json:"token"`
}

// SetServicePermissionsMethod is for use by services.
// This sets the client roles that are allowed to use the service.
// This fails if the client is not a service.
const SetServicePermissionsMethod = "setServicePermissions"

type SetServicePermissionsArgs struct {
	// The service capability to set
	Capability string `json:"capability"`
	// The roles that can use the capability
	Roles []string `json:"roles"`
}

// UpdateNameMethod requests changing the display name of the current client
const UpdateNameMethod = "updateName"

type UpdateNameArgs struct {
	NewName string `json:"newName"`
}

// UpdatePasswordMethod requests changing the password of the current client
const UpdatePasswordMethod = "updatePassword"

type UpdatePasswordArgs struct {
	NewPassword string `json:"newPassword"`
}

// UpdatePubKeyMethod requests changing the public key on file of the current client.
// The public key is used in token validation and generation.
// This takes effect immediately. Existing connection must be closed and re-established.
const UpdatePubKeyMethod = "updatePubKey"

type UpdatePubKeyArgs struct {
	NewPubKey string `json:"newPubKey"`
}
