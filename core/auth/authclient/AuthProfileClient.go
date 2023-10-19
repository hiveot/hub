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
		cl.agentID, cl.capID, auth.GetProfileMethod, nil, &resp)
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
		cl.agentID, cl.capID, auth.NewTokenMethod, &req, &resp)
	return resp.Token, err
}

// RefreshToken a short-lived authentication token.
func (cl *AuthProfileClient) RefreshToken() (authToken string, err error) {
	resp := auth.RefreshTokenResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.RefreshTokenMethod, nil, &resp)
	return resp.NewToken, err
}

// SetServicePermissions for use by services. Set the roles allowed to
// use the service. This is only for use by clients that are services.
func (cl *AuthProfileClient) SetServicePermissions(capID string, roles []string) error {
	args := auth.SetServicePermissionsArgs{
		Capability: capID,
		Roles:      roles,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.SetServicePermissionsMethod, &args, nil)
	return err
}

// UpdateName updates a client's display name
func (cl *AuthProfileClient) UpdateName(newName string) error {
	args := auth.UpdateNameArgs{
		NewName: newName,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdateNameMethod, &args, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (cl *AuthProfileClient) UpdatePassword(newPassword string) error {
	args := auth.UpdatePasswordArgs{
		NewPassword: newPassword,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdatePasswordMethod, &args, nil)
	return err
}

// UpdatePubKey updates the user's public key and close the connection.
// This takes effect immediately. The client must reconnect to continue.
func (cl *AuthProfileClient) UpdatePubKey(newPubKey string) error {
	args := auth.UpdatePubKeyArgs{
		NewPubKey: newPubKey,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdatePubKeyMethod, &args, nil)

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
