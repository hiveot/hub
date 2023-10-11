package authclient

import (
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
)

// AuthProfileClient is a marshaller for auth messages using a provided hub connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type AuthProfileClient struct {
	// ID of the authn service agent
	agentID string
	// capability to invoke
	capID string
	// connection client
	hc hubclient.IHubClient
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *AuthProfileClient) GetProfile() (profile auth.ClientProfile, err error) {
	resp := auth.GetProfileResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.GetProfileReq, nil, &resp)
	return resp.Profile, err
}

// NewToken obtains an auth token based on loginID and password
// The user must have a public key set (using updatePubKey)
func (cl *AuthProfileClient) NewToken(password string) (authToken string, err error) {
	req := auth.NewTokenArgs{
		Password: password,
	}
	resp := auth.NewTokenResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.NewTokenReq, &req, &resp)
	return resp.Token, err
}

// Refresh a short-lived authentication token.
func (cl *AuthProfileClient) Refresh() (authToken string, err error) {
	resp := auth.RefreshResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.RefreshTokenReq, nil, &resp)
	return resp.NewToken, err
}

// UpdateName updates a client's display name
func (cl *AuthProfileClient) UpdateName(newName string) error {
	req := auth.UpdateNameArgs{
		NewName: newName,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdateNameReq, &req, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (cl *AuthProfileClient) UpdatePassword(newPassword string) error {
	req := auth.UpdatePasswordArgs{
		NewPassword: newPassword,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdatePasswordReq, &req, nil)
	return err
}

// UpdatePubKey updates the user's public key and close the connection.
// This takes effect immediately. The client must reconnect to continue.
func (cl *AuthProfileClient) UpdatePubKey(newPubKey string) error {
	req := auth.UpdatePubKeyArgs{
		NewPubKey: newPubKey,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdatePubKeyReq, &req, nil)

	// TBD: as the connection is no longer valid, might as well disconnect it to avoid confusion.
	//cl.hc.Disconnect()
	return err
}

// NewAuthProfileClient returns an auth client for managing a user's profile
//
//	hc is the hub client connection to use
func NewAuthProfileClient(hc hubclient.IHubClient) *AuthProfileClient {
	cl := AuthProfileClient{
		hc:      hc,
		agentID: auth.AuthServiceName,
		capID:   auth.AuthProfileCapability,
	}
	return &cl
}
