// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/runtime/api"
)

type AuthnAdminClient struct {
	mt api.IMessageTransport
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
	err = svc.mt(api.AuthnAdminThingID, api.AddClientMethod, &args, nil)
	return err
}

// GetClientProfile request a client's profile
func (svc *AuthnAdminClient) GetClientProfile(clientID string) (api.ClientProfile, error) {
	args := api.GetClientProfileArgs{ClientID: clientID}
	resp := api.GetProfileResp{}
	err := svc.mt(api.AuthnAdminThingID, api.GetClientProfileMethod, &args, &resp)
	return resp.Profile, err
}

// GetProfiles request the list of known client profiles
func (svc *AuthnAdminClient) GetProfiles() ([]api.ClientProfile, error) {
	resp := api.GetProfilesResp{}
	err := svc.mt(api.AuthnAdminThingID, api.GetProfilesMethod, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client from the system
// After removal the client is no longer able to login.
// Existing login tokens remain valid until they expire
func (svc *AuthnAdminClient) RemoveClient(clientID string) error {
	args := api.RemoveClientArgs{ClientID: clientID}
	err := svc.mt(api.AuthnAdminThingID, api.RemoveClientMethod, &args, nil)
	return err
}

// UpdateClientProfile request update of a client's profile
func (svc *AuthnAdminClient) UpdateClientProfile(profile api.ClientProfile) error {
	args := api.UpdateClientProfileArgs{Profile: profile}
	err := svc.mt(api.AuthnAdminThingID, api.UpdateClientProfileMethod, &args, nil)
	return err
}

// UpdateClientPassword request update of a client's password
func (svc *AuthnAdminClient) UpdateClientPassword(clientID string, password string) error {
	args := api.UpdateClientPasswordArgs{ClientID: clientID, Password: password}
	err := svc.mt(api.AuthnAdminThingID, api.UpdateClientPasswordMethod, &args, nil)
	return err
}

func NewAuthnAdminClient(mt api.IMessageTransport) *AuthnAdminClient {
	cl := AuthnAdminClient{mt: mt}
	return &cl
}