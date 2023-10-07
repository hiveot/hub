package authclient

import (
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
)

// AuthClientsClient is a message (de)serializer for managing clients.
// This uses the default serializer 'ser' to marshal and unmarshal messages.
type AuthClientsClient struct {
	// ID of the authn service
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an rpc request to the auth service
func (cl *AuthClientsClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}
	data, err := cl.hc.PubServiceRPC(
		cl.serviceID, auth2.AuthManageClientsCapability, action, msg)
	if err != nil {
		return err
	}
	if data.ErrorReply != nil {
		return data.ErrorReply
	}
	err = cl.hc.ParseResponse(data.Payload, resp)

	return err
}

// AddDevice adds an IoT device and generates an authentication token
func (cl *AuthClientsClient) AddDevice(
	deviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddDevice", "deviceID", deviceID)
	req := auth2.AddDeviceArgs{
		DeviceID:    deviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := auth2.AddDeviceResp{}
	err := cl.pubReq(auth2.AddDeviceReq, &req, &resp)
	return resp.Token, err
}

// AddService adds a service.
func (cl *AuthClientsClient) AddService(
	serviceID string, displayName string, pubKey string) (string, error) {

	slog.Info("AddService", "serviceID", serviceID)
	req := auth2.AddServiceArgs{
		ServiceID:   serviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
	}
	resp := auth2.AddServiceResp{}
	err := cl.pubReq(auth2.AddServiceReq, &req, &resp)
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
	req := auth2.AddUserArgs{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
		Role:        role,
	}
	resp := auth2.AddUserResp{}
	err := cl.pubReq(auth2.AddUserReq, &req, &resp)
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (cl *AuthClientsClient) GetCount() (n int, err error) {
	resp := auth2.GetCountResp{}
	err = cl.pubReq(auth2.GetCountReq, nil, &resp)
	return resp.N, err
}

// GetAuthClientList provides a list of clients to apply to the message server
//func (mngAuthn *AuthClientsClient) GetAuthClientList() []msgserver.AuthClient {
//	//FIXME
//}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (cl *AuthClientsClient) GetProfile(clientID string) (profile auth2.ClientProfile, err error) {
	req := auth2.GetClientProfileArgs{
		ClientID: clientID,
	}
	resp := auth2.GetProfileResp{}
	err = cl.pubReq(auth2.GetProfileReq, &req, &resp)
	return resp.Profile, err
}

// GetProfiles provide a list of known clients and their info.
// The caller must be an administrator or service.
func (cl *AuthClientsClient) GetProfiles() (profiles []auth2.ClientProfile, err error) {
	resp := auth2.GetProfilesResp{}
	err = cl.pubReq(auth2.GetProfilesReq, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (cl *AuthClientsClient) RemoveClient(clientID string) error {
	req := auth2.RemoveClientArgs{
		ClientID: clientID,
	}
	err := cl.pubReq(auth2.RemoveClientReq, &req, nil)
	return err
}

// UpdateClient updates a client's profile
func (cl *AuthClientsClient) UpdateClient(clientID string, prof auth2.ClientProfile) error {
	req := &auth2.UpdateClientArgs{
		ClientID: clientID,
		Profile:  prof,
	}
	err := cl.pubReq(auth2.UpdateClientReq, req, nil)
	return err
}

// UpdateClientPassword updates a client's password
func (cl *AuthClientsClient) UpdateClientPassword(clientID string, newPass string) error {
	req := &auth2.UpdateClientPasswordArgs{
		ClientID: clientID,
		Password: newPass,
	}
	err := cl.pubReq(auth2.UpdateClientPasswordReq, req, nil)
	return err
}

// UpdateClientRole updates a client's role
func (cl *AuthClientsClient) UpdateClientRole(clientID string, newRole string) error {
	req := &auth2.UpdateClientRoleArgs{
		ClientID: clientID,
		Role:     newRole,
	}
	err := cl.pubReq(auth2.UpdateClientRoleReq, req, nil)
	return err
}

// NewAuthClientsClient returns an authn client management client
//
//	hc is the hub client connection to use
func NewAuthClientsClient(hc hubclient.IHubClient) auth2.IAuthnManageClients {

	cl := AuthClientsClient{
		hc:        hc,
		serviceID: auth2.AuthServiceName,
	}
	return &cl
}
