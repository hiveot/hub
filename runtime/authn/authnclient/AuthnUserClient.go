// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
)

// AuthnUserClient is the client side (un)marshaller for user messages.
// This contains messaging serialization methods for use by end-users including
// agents, services and regular users.
// The main functions are to login, refresh tokens and change name and password.
type AuthnUserClient struct {
	hc hubclient.IHubClient
}

func (svc *AuthnUserClient) GetProfile() (api.ClientProfile, error) {
	resp := api.GetProfileResp{}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.GetProfileMethod, nil, &resp)
	_ = stat
	return resp.Profile, err
}

// FIXME: use a password hash based on a server nonce
func (svc *AuthnUserClient) Login(clientID string, password string) (token string, err error) {
	req := api.LoginArgs{ClientID: clientID, Password: password}
	resp := api.LoginResp{}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.LoginMethod, req, &resp)
	_ = stat
	return resp.Token, err
}

// RefreshToken requests a new token based on the old token
func (svc *AuthnUserClient) RefreshToken(oldToken string) (newToken string, err error) {
	req := api.RefreshTokenArgs{OldToken: oldToken}
	resp := api.RefreshTokenResp{}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.RefreshTokenMethod, req, &resp)
	_ = stat
	return resp.Token, err
}

// UpdateName change the client profile name
func (svc *AuthnUserClient) UpdateName(newName string) error {
	req := api.UpdateNameArgs{NewName: newName}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.UpdateNameMethod, req, nil)
	_ = stat
	return err
}

// UpdatePassword change the client password
// FIXME: encrypt use a password hash based on server nonce
func (svc *AuthnUserClient) UpdatePassword(password string) error {
	req := api.UpdatePasswordArgs{NewPassword: password}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.UpdatePasswordMethod, req, nil)
	_ = stat
	return err
}

// UpdatePubKey updates the client's public key
// Public keys are used in token generation and verification during login.
func (svc *AuthnUserClient) UpdatePubKey(clientID string, pubKeyPem string) error {
	req := api.UpdatePubKeyArgs{PubKeyPem: pubKeyPem}
	stat, err := svc.hc.Rpc(nil, api.AuthnUserThingID, api.UpdatePubKeyMethod, req, nil)
	_ = stat
	return err
}

// NewAuthnUserClient creates a new instance of a client side messaging wrapper
// for communicating with the Authn client service.
func NewAuthnUserClient(hc hubclient.IHubClient) *AuthnUserClient {
	cl := AuthnUserClient{hc: hc}
	return &cl
}
