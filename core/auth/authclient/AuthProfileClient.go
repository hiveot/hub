package authclient

import (
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
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

	data, err := cl.hc.PubServiceRPC(cl.serviceID, auth2.AuthProfileCapability, action, msg)
	if err != nil {
		return err
	}
	if data.ErrorReply != nil {
		return data.ErrorReply
	}
	err = cl.hc.ParseResponse(data.Payload, resp)
	return err
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *AuthProfileClient) GetProfile() (profile auth2.ClientProfile, err error) {
	resp := auth2.GetProfileResp{}
	err = cl.pubReq(auth2.GetProfileReq, nil, &resp)
	return resp.Profile, err
}

// NewToken obtains an auth token based on loginID and password
// The user must have a public key set (using updatePubKey)
func (cl *AuthProfileClient) NewToken(password string) (authToken string, err error) {
	req := auth2.NewTokenArgs{
		Password: password,
	}
	resp := auth2.NewTokenResp{}
	err = cl.pubReq(auth2.NewTokenReq, &req, &resp)
	return resp.Token, err
}

// Refresh a short-lived authentication token.
func (cl *AuthProfileClient) Refresh() (authToken string, err error) {
	resp := auth2.RefreshResp{}
	err = cl.pubReq(auth2.RefreshTokenReq, nil, &resp)
	return resp.NewToken, err
}

// UpdateName updates a client's display name
func (cl *AuthProfileClient) UpdateName(newName string) error {
	req := auth2.UpdateNameArgs{
		NewName: newName,
	}
	err := cl.pubReq(auth2.UpdateNameReq, &req, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (cl *AuthProfileClient) UpdatePassword(newPassword string) error {
	req := auth2.UpdatePasswordArgs{
		NewPassword: newPassword,
	}
	err := cl.pubReq(auth2.UpdatePasswordReq, &req, nil)
	return err
}

// UpdatePubKey updates the user's public key and close the connection.
// This takes effect immediately. The client must reconnect to continue.
func (cl *AuthProfileClient) UpdatePubKey(newPubKey string) error {
	req := auth2.UpdatePubKeyArgs{
		NewPubKey: newPubKey,
	}
	err := cl.pubReq(auth2.UpdatePubKeyReq, &req, nil)
	// as the connection is no longer valid, might as well disconnect it to avoid confusion.
	//cl.hc.Disconnect()
	return err
}

// NewAuthProfileClient returns an auth client for managing a user's profile
//
//	hc is the hub client connection to use
func NewAuthProfileClient(hc hubclient.IHubClient) *AuthProfileClient {
	cl := AuthProfileClient{
		hc:        hc,
		serviceID: auth2.AuthServiceName,
	}
	return &cl
}
