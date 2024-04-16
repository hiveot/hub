// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/runtime/api"
)

// AuthnClientClient is the messaging rpc for the authentication client service
// This contains messaging serialization methods for use by end-users.
// The main functions are to login, refresh tokens and change name and password.
type AuthnClientClient struct {
	pm api.IPostActionMessage
}

func (svc *AuthnClientClient) GetMyProfile() (api.ClientProfile, error) {
	resp := api.GetProfileResp{}
	err := svc.pm(api.AuthnClientServiceID, api.GetProfileMethod, nil, &resp)
	return resp.Profile, err
}

// FIXME: use a password hash based on a server nonce
func (svc *AuthnClientClient) Login(clientID string, password string) (token string, err error) {
	req := api.LoginArgs{ClientID: clientID, Password: password}
	resp := api.LoginResp{}
	err = svc.pm(api.AuthnClientServiceID, api.LoginMethod, req, &resp)
	return resp.Token, err
}

// RefreshToken requests a new token based on the old token
func (svc *AuthnClientClient) RefreshToken(clientID string, oldToken string) (newToken string, err error) {
	req := api.RefreshTokenArgs{ClientID: clientID, OldToken: oldToken}
	resp := api.RefreshTokenResp{}
	err = svc.pm(api.AuthnClientServiceID, api.RefreshTokenMethod, req, &resp)
	return resp.Token, err
}

// UpdateName change the client profile name
func (svc *AuthnClientClient) UpdateName(newName string) error {
	req := api.UpdateNameArgs{NewName: newName}
	err := svc.pm(api.AuthnClientServiceID, api.UpdateNameMethod, req, nil)
	return err
}

// UpdatePassword change the client password
// FIXME: encrypt use a password hash based on server nonce
func (svc *AuthnClientClient) UpdatePassword(clientID string, password string) error {
	req := api.UpdatePasswordArgs{ClientID: clientID, NewPassword: password}
	err := svc.pm(api.AuthnClientServiceID, api.UpdatePasswordMethod, req, nil)
	return err
}

// UpdatePubKey updates the client's public key
// Public keys are used in token generation and verification during login.
func (svc *AuthnClientClient) UpdatePubKey(clientID string, pubKeyPem string) error {
	req := api.UpdatePubKeyArgs{ClientID: clientID, PubKeyPem: pubKeyPem}
	err := svc.pm(api.AuthnClientServiceID, api.UpdatePubKeyMethod, req, nil)
	return err
}

// NewAuthnClientClient creates a new instance of a client side messaging wrapper
// for communicating with the Authn client service.
func NewAuthnClientClient(pm api.IPostActionMessage) *AuthnClientClient {
	cl := AuthnClientClient{pm: pm}
	return &cl
}
