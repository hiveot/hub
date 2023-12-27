package authclient

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

// ManageClients is a message (de)serializer for managing clients.
// This uses the default serializer 'ser' to marshal and unmarshal messages.
type ManageClients struct {
	// ID of the authn service agent
	agentID string
	// capability to invoke
	capID string
	// connection client
	hc *hubclient.HubClient
}

// AddDevice adds an IoT device and generates an authentication token
func (cl *ManageClients) AddDevice(
	deviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddDevice", "deviceID", deviceID)
	req := authapi.AddDeviceArgs{
		DeviceID:    deviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := authapi.AddDeviceResp{}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.AddDeviceMethod, &req, &resp)
	return resp.Token, err
}

// AddService adds a service with the given serviceID
// To generate a key/token use the ProfileClient
func (cl *ManageClients) AddService(
	serviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddService", "serviceID", serviceID)
	req := authapi.AddServiceArgs{
		ServiceID:   serviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := authapi.AddServiceResp{}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.AddServiceMethod, &req, &resp)
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
func (cl *ManageClients) AddUser(
	userID string, displayName string, password string, pubKey string, role string) (string, error) {

	slog.Info("AddUser", "userID", userID)
	req := authapi.AddUserArgs{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
		Role:        role,
	}
	resp := authapi.AddUserResp{}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.AddUserMethod, &req, &resp)
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (cl *ManageClients) GetCount() (n int, err error) {
	resp := authapi.GetCountResp{}
	err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.GetCountMethod, nil, &resp)
	return resp.N, err
}

// GetAuthClientList provides a list of clients to apply to the message server
//func (mngAuthn *ManageClients) GetAuthClientList() []msgserver.AuthClient {
//	//FIXME
//}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *ManageClients) GetProfile(clientID string) (profile authapi.ClientProfile, err error) {
	req := authapi.GetClientProfileArgs{
		ClientID: clientID,
	}
	resp := authapi.GetProfileResp{}
	err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.GetProfilesMethod, &req, &resp)
	return resp.Profile, err
}

// GetProfiles provide a list of known clients and their info.
// The caller must be an administrator or service.
func (cl *ManageClients) GetProfiles() (profiles []authapi.ClientProfile, err error) {
	resp := authapi.GetProfilesResp{}
	err = cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.GetProfilesMethod, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (cl *ManageClients) RemoveClient(clientID string) error {
	req := authapi.RemoveClientArgs{
		ClientID: clientID,
	}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.RemoveClientMethod, &req, nil)
	return err
}

// SetClientPassword sets a new password for a client
func (cl *ManageClients) SetClientPassword(clientID string, newPass string) error {
	req := &authapi.SetClientPasswordArgs{
		ClientID: clientID,
		Password: newPass,
	}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.SetClientPasswordMethod, &req, nil)
	return err
}

// UpdateClient updates a client's profile
func (cl *ManageClients) UpdateClient(clientID string, prof authapi.ClientProfile) error {
	req := &authapi.UpdateClientArgs{
		ClientID: clientID,
		Profile:  prof,
	}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.UpdateClientMethod, &req, nil)
	return err
}

// UpdateClientRole updates a client's role
func (cl *ManageClients) UpdateClientRole(clientID string, newRole string) error {
	req := &authapi.UpdateClientRoleArgs{
		ClientID: clientID,
		Role:     newRole,
	}
	err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, authapi.UpdateClientRoleMethod, &req, nil)
	return err
}

// NewManageClients returns an authn client management client
//
//	hc is the hub client connection to use
func NewManageClients(hc *hubclient.HubClient) *ManageClients {

	cl := ManageClients{
		hc:      hc,
		agentID: authapi.AuthServiceName,
		capID:   authapi.AuthManageClientsCapability,
	}
	return &cl
}
