// Package authnclient with a golang client to access the authn service methods
package authnclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

type AuthnAdminClient struct {
	//mt api.IMessageTransport
	hc hubclient.IHubClient
	// the authn digital twin thingID to communicate with
	dtThingID string
}

// AddUser adds a new user client
func (svc *AuthnAdminClient) AddUser(
	clientID string, displayName string, password string) (err error) {

	args := api.AddUserArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		Password:    password,
	}
	err = svc.hc.Rpc(svc.dtThingID, api.AddUserMethod, &args, nil)
	return err
}

// AddAgent adds a device, service or admin user client and generates a token file
func (svc *AuthnAdminClient) AddAgent(
	clientType api.ClientType, clientID string, displayName string, pubKey string) (token string, err error) {

	args := api.AddAgentArgs{
		ClientType:  clientType,
		ClientID:    clientID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := api.AddAgentResp{}
	err = svc.hc.Rpc(svc.dtThingID, api.AddAgentMethod, &args, &resp)
	return resp.Token, err
}

// GetClientProfile request a client's profile
func (svc *AuthnAdminClient) GetClientProfile(clientID string) (api.ClientProfile, error) {
	args := api.GetClientProfileArgs{ClientID: clientID}
	resp := api.GetProfileResp{}
	err := svc.hc.Rpc(svc.dtThingID, api.GetClientProfileMethod, &args, &resp)
	return resp.Profile, err
}

// GetProfiles request the list of known client profiles
func (svc *AuthnAdminClient) GetProfiles() ([]api.ClientProfile, error) {
	resp := api.GetProfilesResp{}
	err := svc.hc.Rpc(svc.dtThingID, api.GetProfilesMethod, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client from the system
// After removal the client is no longer able to login.
// Existing login tokens remain valid until they expire
func (svc *AuthnAdminClient) RemoveClient(clientID string) error {
	args := api.RemoveClientArgs{ClientID: clientID}
	err := svc.hc.Rpc(svc.dtThingID, api.RemoveClientMethod, &args, nil)
	return err
}

// NewAuthToken request a new authentication token for an agent or service
func (svc *AuthnAdminClient) NewAuthToken(clientID string, validitySec int) (token string, err error) {
	args := api.NewAuthTokenArgs{ClientID: clientID, ValiditySec: validitySec}
	resp := api.NewAuthTokenResp{}
	err = svc.hc.Rpc(svc.dtThingID, api.NewAuthTokenMethod, &args, &resp)
	return resp.Token, err
}

// SetClientPassword request update of a client's password
func (svc *AuthnAdminClient) SetClientPassword(clientID string, password string) error {
	args := api.SetClientPasswordArgs{ClientID: clientID, Password: password}
	err := svc.hc.Rpc(svc.dtThingID, api.SetClientPasswordMethod, &args, nil)
	return err
}

// UpdateClientProfile request update of a client's profile
func (svc *AuthnAdminClient) UpdateClientProfile(profile api.ClientProfile) error {
	args := api.UpdateClientProfileArgs{Profile: profile}
	err := svc.hc.Rpc(svc.dtThingID, api.UpdateClientProfileMethod, &args, nil)
	return err
}

func NewAuthnAdminClient(hc hubclient.IHubClient) *AuthnAdminClient {
	cl := AuthnAdminClient{
		hc:        hc,
		dtThingID: things.MakeDigiTwinThingID(api.AuthnAgentID, api.AuthnAdminServiceID),
	}
	return &cl
}
