package authclient

import (
	auth "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

// AuthClientsClient is a message (de)serializer for managing clients.
// This uses the default serializer 'ser' to marshal and unmarshal messages.
type AuthClientsClient struct {
	// ID of the authn service agent
	agentID string
	// capability to invoke
	capID string
	// connection client
	hc hubclient.IHubClient
}

// AddDevice adds an IoT device and generates an authentication token
func (cl *AuthClientsClient) AddDevice(
	deviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddDevice", "deviceID", deviceID)
	req := auth.AddDeviceArgs{
		DeviceID:    deviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := auth.AddDeviceResp{}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.AddDeviceMethod, &req, &resp)
	return resp.Token, err
}

// AddService adds a service.
func (cl *AuthClientsClient) AddService(
	serviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddService", "serviceID", serviceID)
	req := auth.AddServiceArgs{
		ServiceID:   serviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := auth.AddServiceResp{}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.AddServiceMethod, &req, &resp)
	return resp.Token, err
}

// AddUser adds a user.
// The caller must be an administrator or service.
// If the userID already exists then an error is returned
//
//	userID is the login ID of the user, typically their email
//	name of the user for presentation
//	password the user can login with if their token has expired.
//	pubKey is the user's public key string, needed to connect with JWT
func (cl *AuthClientsClient) AddUser(
	userID string, displayName string, password string, pubKey string, role string) (string, error) {

	slog.Info("AddUser", "userID", userID)
	req := auth.AddUserArgs{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
		Role:        role,
	}
	resp := auth.AddUserResp{}
	ar, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.AddUserMethod, &req, &resp)
	_ = ar
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (cl *AuthClientsClient) GetCount() (n int, err error) {
	resp := auth.GetCountResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.GetCountMethod, nil, &resp)
	return resp.N, err
}

// GetAuthClientList provides a list of clients to apply to the message server
//func (mngAuthn *AuthClientsClient) GetAuthClientList() []msgserver.AuthClient {
//	//FIXME
//}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *AuthClientsClient) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	req := auth.GetClientProfileArgs{
		ClientID: clientID,
	}
	resp := auth.GetProfileResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.GetProfilesMethod, &req, &resp)
	return resp.Profile, err
}

// GetProfiles provide a list of known clients and their info.
// The caller must be an administrator or service.
func (cl *AuthClientsClient) GetProfiles() (profiles []auth.ClientProfile, err error) {
	resp := auth.GetProfilesResp{}
	_, err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.GetProfilesMethod, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (cl *AuthClientsClient) RemoveClient(clientID string) error {
	req := auth.RemoveClientArgs{
		ClientID: clientID,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.RemoveClientMethod, &req, nil)
	return err
}

// UpdateClient updates a client's profile
func (cl *AuthClientsClient) UpdateClient(clientID string, prof auth.ClientProfile) error {
	req := &auth.UpdateClientArgs{
		ClientID: clientID,
		Profile:  prof,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdateClientMethod, &req, nil)
	return err
}

// UpdateClientPassword updates a client's password
func (cl *AuthClientsClient) UpdateClientPassword(clientID string, newPass string) error {
	req := &auth.UpdateClientPasswordArgs{
		ClientID: clientID,
		Password: newPass,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdatePasswordMethod, &req, nil)
	return err
}

// UpdateClientRole updates a client's role
func (cl *AuthClientsClient) UpdateClientRole(clientID string, newRole string) error {
	req := &auth.UpdateClientRoleArgs{
		ClientID: clientID,
		Role:     newRole,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.UpdateClientRoleMethod, &req, nil)
	return err
}

// NewAuthClientsClient returns an authn client management client
//
//	hc is the hub client connection to use
func NewAuthClientsClient(hc hubclient.IHubClient) auth.IAuthnManageClients {

	cl := AuthClientsClient{
		hc:      hc,
		agentID: auth.AuthServiceName,
		capID:   auth.AuthManageClientsCapability,
	}
	return &cl
}
