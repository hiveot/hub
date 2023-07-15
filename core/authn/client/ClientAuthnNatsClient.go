package client

import (
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// ClientAuthn is a marshaller for messaging with the authn service using the hub client connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type ClientAuthn struct {
	// ID of the authn service that handles the requests
	bindingID string
	hc        hub.IHubClient
}

// helper for publishing an action request to the authz service
func (clientAuthn *ClientAuthn) pubReq(action string, msg []byte) ([]byte, error) {
	return clientAuthn.hc.PubAction(clientAuthn.bindingID, authn.ClientAuthnCapability, action, msg)
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (clientAuthn *ClientAuthn) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	req := authn.GetProfileReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.GetProfileAction, msg)
	resp := &authn.GetProfileResp{}
	err = hubclient.ParseResponse(data, err, resp)
	if err == nil {
		profile = resp.Profile
	}
	return profile, err
}

// NewToken obtains an auth token based on loginID and password
func (clientAuthn *ClientAuthn) NewToken(clientID string, password string, pubKey string) (authToken string, err error) {
	req := authn.NewTokenReq{
		ClientID: clientID,
		PubKey:   pubKey,
		Password: password,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.NewTokenAction, msg)
	resp := &authn.NewTokenResp{}
	err = hubclient.ParseResponse(data, err, resp)
	if err == nil {
		authToken = resp.JwtToken
	}
	return authToken, err
}

// Refresh a short-lived authentication token.
func (clientAuthn *ClientAuthn) Refresh(clientID string, oldToken string) (authToken string, err error) {
	req := authn.RefreshReq{
		ClientID: clientID,
		OldToken: oldToken,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.RefreshAction, msg)
	resp := &authn.RefreshResp{}
	err = hubclient.ParseResponse(data, err, resp)
	if err == nil {
		authToken = resp.JwtToken
	}
	return authToken, err
}

// UpdateName updates a client's display name
func (clientAuthn *ClientAuthn) UpdateName(clientID string, newName string) error {
	req := authn.UpdateNameReq{
		ClientID: clientID,
		NewName:  newName,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.UpdateNameAction, msg)
	err = hubclient.ParseResponse(data, err, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (clientAuthn *ClientAuthn) UpdatePassword(clientID string, newPassword string) error {
	req := authn.UpdatePasswordReq{
		ClientID:    clientID,
		NewPassword: newPassword,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.UpdatePasswordAction, msg)
	err = hubclient.ParseResponse(data, err, nil)
	return err
}

// NewClientAuthn returns an authn client for the given hubclient connection
//
//	bindingID is the ID of the authn service. Use "" for default.
func NewClientAuthn(bindingID string, hc hub.IHubClient) *ClientAuthn {
	if bindingID == "" {
		bindingID = authn.AuthnServiceName
	}
	cl := ClientAuthn{
		hc:        hc,
		bindingID: bindingID,
	}
	return &cl
}
