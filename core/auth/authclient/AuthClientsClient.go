package authclient

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
	"time"
)

// AuthClientsClient is a message (de)serializer for managing clients.
// This uses the default serializer to marshal and unmarshal messages.
type AuthClientsClient struct {
	// ID of the authn service
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (cl *AuthClientsClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}
	data, err := cl.hc.PubServiceAction(
		cl.serviceID, auth.AuthManageClientsCapability, action, msg)

	err = cl.hc.ParseResponse(data, err, resp)

	return err
}

// AddDevice adds an IoT device and generates an authentication token
func (cl *AuthClientsClient) AddDevice(
	deviceID string, displayName string, pubKey string, tokenValidity time.Duration) (string, error) {

	slog.Info("AddDevice", "deviceID", deviceID)
	req := auth.AddDeviceReq{
		DeviceID:      deviceID,
		DisplayName:   displayName,
		PubKey:        pubKey,
		TokenValidity: tokenValidity,
	}
	resp := auth.AddDeviceResp{}
	err := cl.pubReq(auth.AddDeviceAction, &req, &resp)
	return resp.Token, err
}

// AddService adds a service.
func (cl *AuthClientsClient) AddService(
	serviceID string, displayName string, pubKey string, tokenValidity time.Duration) (string, error) {

	slog.Info("AddService", "serviceID", serviceID)
	req := auth.AddServiceReq{
		ServiceID:   serviceID,
		DisplayName: displayName,
		PubKey:      pubKey,

		TokenValidity: tokenValidity,
	}
	resp := auth.AddServiceResp{}
	err := cl.pubReq(auth.AddServiceAction, &req, &resp)
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
	req := auth.AddUserReq{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
		Role:        role,
	}
	resp := auth.AddUserResp{}
	err := cl.pubReq(auth.AddUserAction, &req, &resp)
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (cl *AuthClientsClient) GetCount() (n int, err error) {
	resp := auth.GetCountResp{}
	err = cl.pubReq(auth.GetCountAction, nil, &resp)
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
	req := auth.GetProfileReq{
		ClientID: clientID,
	}
	resp := auth.GetProfileResp{}
	err = cl.pubReq(auth.GetProfileAction, &req, &resp)
	return resp.Profile, err
}

// GetProfiles provide a list of known clients and their info.
// The caller must be an administrator or service.
func (cl *AuthClientsClient) GetProfiles() (profiles []auth.ClientProfile, err error) {
	resp := auth.GetProfilesResp{}
	err = cl.pubReq(auth.GetProfilesAction, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (cl *AuthClientsClient) RemoveClient(clientID string) error {
	req := auth.RemoveClientReq{
		ClientID: clientID,
	}
	err := cl.pubReq(auth.RemoveClientAction, &req, nil)
	return err
}

// UpdateClient updates a client's profile
func (cl *AuthClientsClient) UpdateClient(clientID string, prof auth.ClientProfile) error {
	req := &auth.UpdateClientReq{
		ClientID: clientID,
		Profile:  prof,
	}
	err := cl.pubReq(auth.UpdateClientAction, req, nil)
	return err
}

// NewAuthClientsClient returns an authn client management client
func NewAuthClientsClient(hc hubclient.IHubClient) auth.IAuthnManageClients {
	bindingID := auth.AuthServiceName

	cl := AuthClientsClient{
		hc:        hc,
		serviceID: bindingID,
	}
	return &cl
}
