package authclient

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// AuthProfileClient is a marshaller for auth messages using a provided hub connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type AuthProfileClient struct {
	// ID of the authn service that handles the requests
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (cl *AuthProfileClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}

	data, err := cl.hc.PubServiceAction(cl.serviceID, auth.AuthProfileCapability, action, msg)
	err = cl.hc.ParseResponse(data, err, resp)
	return err
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *AuthProfileClient) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	req := auth.GetProfileReq{
		ClientID: clientID,
	}
	resp := auth.GetProfileResp{}
	err = cl.pubReq(auth.GetProfileAction, &req, &resp)
	return resp.Profile, err
}

// NewToken obtains an auth token based on loginID and password
// The user must have a public key set (using updatePubKey)
func (cl *AuthProfileClient) NewToken(clientID string, password string) (authToken string, err error) {
	req := auth.NewTokenReq{
		ClientID: clientID,
		Password: password,
	}
	resp := auth.NewTokenResp{}
	err = cl.pubReq(auth.NewTokenAction, &req, &resp)
	return resp.Token, err
}

// Refresh a short-lived authentication token.
func (cl *AuthProfileClient) Refresh(clientID string, oldToken string) (authToken string, err error) {
	req := auth.RefreshReq{
		ClientID: clientID,
		OldToken: oldToken,
	}
	resp := auth.RefreshResp{}
	err = cl.pubReq(auth.RefreshAction, &req, &resp)
	return resp.NewToken, err
}

// UpdateName updates a client's display name
func (cl *AuthProfileClient) UpdateName(clientID string, newName string) error {
	req := auth.UpdateNameReq{
		ClientID: clientID,
		NewName:  newName,
	}
	err := cl.pubReq(auth.UpdateNameAction, &req, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (cl *AuthProfileClient) UpdatePassword(clientID string, newPassword string) error {
	req := auth.UpdatePasswordReq{
		ClientID:    clientID,
		NewPassword: newPassword,
	}
	err := cl.pubReq(auth.UpdatePasswordAction, &req, nil)
	return err
}

// UpdatePubKey updates the user's public key and close the connection.
// This takes effect immediately. The client must reconnect to continue.
func (cl *AuthProfileClient) UpdatePubKey(clientID string, newPubKey string) error {
	req := auth.UpdatePubKeyReq{
		ClientID:  clientID,
		NewPubKey: newPubKey,
	}
	err := cl.pubReq(auth.UpdatePubKeyAction, &req, nil)
	// as the connection is no longer valid, might as well disconnect it to avoid confusion.
	cl.hc.Disconnect()
	return err
}

// NewAuthProfileClient returns an auth client for managing a user's profile
//
//	hc is the hub client connection to use
func NewAuthProfileClient(hc hubclient.IHubClient) *AuthProfileClient {
	serviceID := auth.AuthServiceName
	cl := AuthProfileClient{
		hc:        hc,
		serviceID: serviceID,
	}
	return &cl
}
