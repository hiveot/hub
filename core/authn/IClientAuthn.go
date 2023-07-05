package authn

import "github.com/hiveot/hub/api/go/hub"

const NewTokenAction = "newToken"

type NewTokenReq struct {
	// ClientID is the publisherID of the action
	// Password to verify identity
	Password string `json:"password"`
	// Client public key for the token
	PubKey string `json:"pubKey"`
}
type LoginResp struct {
	hub.ErrorMessage
	JwtToken string `json:"jwtToken"`
}

const RefreshAction = "refresh"

type RefreshReq struct {
	// This uses the publisherID as clientID to verify
	OldToken string `json:"oldToken"`
}
type RefreshResp struct {
	hub.ErrorMessage
	AuthToken string `json:"authToken"`
}

const UpdateNameAction = "updateName"

type UpdateNameReq struct {
	ClientID string `json:"clientID"`
	NewName  string `json:"newName"`
}

const UpdatePasswordAction = "updatePassword"

type UpdatePasswordReq struct {
	ClientID    string `json:"clientID"`
	NewPassword string `json:"newPassword"`
}

// IClientAuthn defines the capabilities for use by authenticating clients
type IClientAuthn interface {
	// NewToken creates a new jwt auth token based on userName, password and public key
	// This returns a short-lived auth token that can be used to connect to the message server
	// The token can be refreshed to extend it without requiring a login password.
	NewToken(clientID string, password string, pubKey string) (jwtToken string, err error)

	// Refresh a short-lived authentication token.
	//
	//  clientID is the userID, deviceID or serviceID whose token to refresh.
	//  oldToken must be a valid token obtained at login or refresh
	//
	// This returns a new short-lived auth token that can be used to authenticate with the hub
	// This fails if the token has expired or does not belong to the clientID
	Refresh(clientID string, oldToken string) (authToken string, err error)

	// UpdateName updates a user's display name
	UpdateName(clientID string, newName string) (err error)

	// UpdatePassword changes the client password
	// Login or Refresh must be called successfully first.
	UpdatePassword(clientID string, newPassword string) error
}
