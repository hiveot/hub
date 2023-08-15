package authnclient

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// AuthnUserClient is a marshaller for authn messages using a provided hub connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type AuthnUserClient struct {
	// ID of the authn service that handles the requests
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (clientAuthn *AuthnUserClient) pubReq(action string, msg []byte) ([]byte, error) {
	return clientAuthn.hc.PubAction(clientAuthn.serviceID, authn.ClientAuthnCapability, action, msg)
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (clientAuthn *AuthnUserClient) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	req := authn.GetProfileReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.GetProfileAction, msg)
	if err != nil {
		return profile, err
	}
	resp := &authn.GetProfileResp{}
	err = clientAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		profile = resp.Profile
	}
	return profile, err
}

// NewToken obtains an auth token based on loginID and password
func (clientAuthn *AuthnUserClient) NewToken(clientID string, password string) (authToken string, err error) {
	req := authn.NewTokenReq{
		ClientID: clientID,
		Password: password,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.NewTokenAction, msg)
	if err != nil {
		return "", err
	}
	resp := &authn.NewTokenResp{}
	err = clientAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		authToken = resp.Token
	}
	return authToken, err
}

// Refresh a short-lived authentication token.
func (clientAuthn *AuthnUserClient) Refresh(clientID string, oldToken string) (authToken string, err error) {
	req := authn.RefreshReq{
		ClientID: clientID,
		OldToken: oldToken,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.RefreshAction, msg)
	if err != nil {
		return "", err
	}
	resp := &authn.RefreshResp{}
	err = clientAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		authToken = resp.JwtToken
	}
	return authToken, err
}

// UpdateName updates a client's display name
func (clientAuthn *AuthnUserClient) UpdateName(clientID string, newName string) error {
	req := authn.UpdateNameReq{
		ClientID: clientID,
		NewName:  newName,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.UpdateNameAction, msg)
	err = clientAuthn.hc.ParseResponse(data, err, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (clientAuthn *AuthnUserClient) UpdatePassword(clientID string, newPassword string) error {
	req := authn.UpdatePasswordReq{
		ClientID:    clientID,
		NewPassword: newPassword,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.UpdatePasswordAction, msg)
	err = clientAuthn.hc.ParseResponse(data, err, nil)
	return err
}

// UpdatePubKey updates the user's public key
// Login or Refresh must be called successfully first.
func (clientAuthn *AuthnUserClient) UpdatePubKey(clientID string, newPubKey string) error {
	req := authn.UpdatePubKeyReq{
		ClientID:  clientID,
		NewPubKey: newPubKey,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.UpdatePubKeyAction, msg)
	err = clientAuthn.hc.ParseResponse(data, err, nil)
	return err
}

// NewAuthnUserClient returns an authn client for the given hubclient connection
//
//	hc is the hub client connection to use
func NewAuthnUserClient(hc hubclient.IHubClient) *AuthnUserClient {
	serviceID := authn.AuthnServiceName
	cl := AuthnUserClient{
		hc:        hc,
		serviceID: serviceID,
	}
	return &cl
}
