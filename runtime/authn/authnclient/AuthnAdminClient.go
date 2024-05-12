// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
)

type AuthnAdminClient struct {
	//mt api.IMessageTransport
	hc hubclient.IHubClient
}

// AddClient adds a new client
func (svc *AuthnAdminClient) AddClient(
	clientType api.ClientType,
	clientID string, displayName string,
	pubKey string, password string) (err error) {
	args := api.AddClientArgs{
		ClientType:  clientType,
		ClientID:    clientID,
		DisplayName: displayName,
		PubKey:      pubKey,
		Password:    password,
	}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.AddClientMethod, &args, nil)
	_ = stat
	return err
}

// GetClientProfile request a client's profile
func (svc *AuthnAdminClient) GetClientProfile(clientID string) (api.ClientProfile, error) {
	args := api.GetClientProfileArgs{ClientID: clientID}
	resp := api.GetProfileResp{}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.GetClientProfileMethod, &args, &resp)
	_ = stat
	return resp.Profile, err
}

// GetProfiles request the list of known client profiles
func (svc *AuthnAdminClient) GetProfiles() ([]api.ClientProfile, error) {
	resp := api.GetProfilesResp{}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.GetProfilesMethod, nil, &resp)
	_ = stat
	return resp.Profiles, err
}

// RemoveClient removes a client from the system
// After removal the client is no longer able to login.
// Existing login tokens remain valid until they expire
func (svc *AuthnAdminClient) RemoveClient(clientID string) error {
	args := api.RemoveClientArgs{ClientID: clientID}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.RemoveClientMethod, &args, nil)
	_ = stat
	return err
}

// UpdateClientProfile request update of a client's profile
func (svc *AuthnAdminClient) UpdateClientProfile(profile api.ClientProfile) error {
	args := api.UpdateClientProfileArgs{Profile: profile}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.UpdateClientProfileMethod, &args, nil)
	_ = stat
	return err
}

// UpdateClientPassword request update of a client's password
func (svc *AuthnAdminClient) UpdateClientPassword(clientID string, password string) error {
	args := api.UpdateClientPasswordArgs{ClientID: clientID, Password: password}
	stat, err := svc.hc.Rpc(nil, api.AuthnAdminThingID, api.UpdateClientPasswordMethod, &args, nil)
	_ = stat
	return err
}

func NewAuthnAdminClient(hc hubclient.IHubClient) *AuthnAdminClient {
	cl := AuthnAdminClient{hc: hc}
	return &cl
}
