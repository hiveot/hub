package authnclient

import (
	"github.com/hiveot/hub/api/go/auth"
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
func (clientAuthn *AuthnUserClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}

	data, err := clientAuthn.hc.PubAction(clientAuthn.serviceID, auth.ClientAuthnCapability, action, msg)
	err = clientAuthn.hc.ParseResponse(data, err, resp)
	return err
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (clientAuthn *AuthnUserClient) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	req := auth.GetProfileReq{
		ClientID: clientID,
	}
	resp := auth.GetProfileResp{}
	err = clientAuthn.pubReq(auth.GetProfileAction, &req, &resp)
	return resp.Profile, err
}

// NewToken obtains an auth token based on loginID and password
// The user must have a public key set (using updatePubKey)
func (clientAuthn *AuthnUserClient) NewToken(clientID string, password string) (authToken string, err error) {
	req := auth.NewTokenReq{
		ClientID: clientID,
		Password: password,
	}
	resp := auth.NewTokenResp{}
	err = clientAuthn.pubReq(auth.NewTokenAction, &req, &resp)
	return resp.Token, err
}

// Refresh a short-lived authentication token.
func (clientAuthn *AuthnUserClient) Refresh(clientID string, oldToken string) (authToken string, err error) {
	req := auth.RefreshReq{
		ClientID: clientID,
		OldToken: oldToken,
	}
	resp := auth.RefreshResp{}
	err = clientAuthn.pubReq(auth.RefreshAction, &req, &resp)
	return resp.NewToken, err
}

// UpdateName updates a client's display name
func (clientAuthn *AuthnUserClient) UpdateName(clientID string, newName string) error {
	req := auth.UpdateNameReq{
		ClientID: clientID,
		NewName:  newName,
	}
	err := clientAuthn.pubReq(auth.UpdateNameAction, &req, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (clientAuthn *AuthnUserClient) UpdatePassword(clientID string, newPassword string) error {
	req := auth.UpdatePasswordReq{
		ClientID:    clientID,
		NewPassword: newPassword,
	}
	err := clientAuthn.pubReq(auth.UpdatePasswordAction, &req, nil)
	return err
}

// UpdatePubKey updates the user's public key.
// This takes effect immediately. Existing connection must be closed and re-established.
// Login or Refresh must be called successfully first.
func (clientAuthn *AuthnUserClient) UpdatePubKey(clientID string, newPubKey string) error {
	req := auth.UpdatePubKeyReq{
		ClientID:  clientID,
		NewPubKey: newPubKey,
	}
	err := clientAuthn.pubReq(auth.UpdatePubKeyAction, &req, nil)
	return err
}

// NewAuthnUserClient returns an authn client for the given hubclient connection
//
//	hc is the hub client connection to use
func NewAuthnUserClient(hc hubclient.IHubClient) *AuthnUserClient {
	serviceID := auth.AuthServiceName
	cl := AuthnUserClient{
		hc:        hc,
		serviceID: serviceID,
	}
	return &cl
}
