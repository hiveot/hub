// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/runtime/api"
)

type AuthnAdminClient struct {
	pm api.IPostActionMessage
}

// AddClient adds a new client
func (svc *AuthnAdminClient) AddClient(
	clientType api.ClientType,
	clientID string, displayName string,
	pubKey string, password string) (err error) {
	req := api.AddClientArgs{
		ClientType:  clientType,
		ClientID:    clientID,
		DisplayName: displayName,
		PubKey:      pubKey,
		Password:    password,
	}
	err = svc.pm(api.AuthnAdminServiceID, api.AddClientMethod, req, nil)
	return err
}

// GetClientProfile request a client's profile
func (svc *AuthnAdminClient) GetClientProfile(clientID string) (api.ClientProfile, error) {
	req := api.GetClientProfileArgs{ClientID: clientID}
	resp := api.GetProfileResp{}
	err := svc.pm(api.AuthnAdminServiceID, api.GetClientProfileMethod, req, &resp)
	return resp.Profile, err
}

// GetProfiles request the list of known client profiles
func (svc *AuthnAdminClient) GetProfiles() ([]api.ClientProfile, error) {
	resp := api.GetProfilesResp{}
	err := svc.pm(api.AuthnAdminServiceID, api.GetProfilesMethod, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client from the system
// After removal the client is no longer able to login.
// Existing login tokens remain valid until they expire
func (svc *AuthnAdminClient) RemoveClient(clientID string) error {
	req := api.RemoveClientArgs{ClientID: clientID}
	err := svc.pm(api.AuthnAdminServiceID, api.RemoveClientMethod, req, nil)
	return err
}

// UpdateClientProfile request update of a client's profile
func (svc *AuthnAdminClient) UpdateClientProfile(profile api.ClientProfile) error {
	args := api.UpdateClientProfileArgs{Profile: profile}
	err := svc.pm(api.AuthnAdminServiceID, api.UpdateClientProfileMethod, args, nil)
	return err
}

// UpdateClientPassword request update of a client's password
func (svc *AuthnAdminClient) UpdateClientPassword(clientID string, password string) error {
	args := api.UpdateClientPasswordArgs{ClientID: clientID, Password: password}
	err := svc.pm(api.AuthnAdminServiceID, api.UpdateClientPasswordMethod, args, nil)
	return err
}

func NewAuthnAdminClient(pm api.IPostActionMessage) *AuthnAdminClient {
	cl := AuthnAdminClient{pm: pm}
	return &cl
}
