package authclient

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/hubclient"
)

// ProfileClient is a marshaller for auth messages using a provided hub connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type ProfileClient struct {
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
func (cl *ProfileClient) GetProfile() (profile authapi.ClientProfile, err error) {
	resp := authapi.GetProfileResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.GetProfileMethod, nil, &resp)
	return resp.Profile, err
}

// NewToken obtains an auth token based on loginID and password
// The user must have a public key set (using updatePubKey)
func (cl *ProfileClient) NewToken(password string) (authToken string, err error) {
	req := authapi.NewTokenArgs{
		Password: password,
	}
	resp := authapi.NewTokenResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.NewTokenMethod, &req, &resp)
	return resp.Token, err
}

// RefreshToken a short-lived authentication token.
func (cl *ProfileClient) RefreshToken() (authToken string, err error) {
	resp := authapi.RefreshTokenResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.RefreshTokenMethod, nil, &resp)
	return resp.Token, err
}

// SetServicePermissions for use by services. Set the roles allowed to
// use the service. This is only for use by clients that are services.
func (cl *ProfileClient) SetServicePermissions(capID string, roles []string) error {
	args := authapi.SetServicePermissionsArgs{
		Capability: capID,
		Roles:      roles,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.SetServicePermissionsMethod, &args, nil)
	return err
}

// UpdateName updates a client's display name
func (cl *ProfileClient) UpdateName(newName string) error {
	args := authapi.UpdateNameArgs{
		NewName: newName,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.UpdateNameMethod, &args, nil)
	return err
}

// UpdatePassword changes the user password
// Login or Refresh must be called successfully first.
func (cl *ProfileClient) UpdatePassword(newPassword string) error {
	args := authapi.UpdatePasswordArgs{
		NewPassword: newPassword,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.UpdatePasswordMethod, &args, nil)
	return err
}

// UpdatePubKey updates the user's public key and close the connection.
// This takes effect immediately. The client must reconnect to continue.
func (cl *ProfileClient) UpdatePubKey(newPubKey string) error {
	args := authapi.UpdatePubKeyArgs{
		NewPubKey: newPubKey,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.UpdatePubKeyMethod, &args, nil)

	// TBD: as the connection is no longer valid, might as well disconnect it to avoid confusion.
	//cl.hc.Disconnect()
	return err
}

// NewProfileClient returns an auth client for managing a user's profile
//
//	hc is the hub client connection to use
func NewProfileClient(hc hubclient.IHubClient) *ProfileClient {
	cl := ProfileClient{
		hc:      hc,
		agentID: authapi.AuthServiceName,
		capID:   authapi.AuthProfileCapability,
	}
	return &cl
}
