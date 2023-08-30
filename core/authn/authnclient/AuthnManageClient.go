package authnclient

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
	"time"
)

// AuthnManageClient is a client for the authn service for administrators
// and services that manage authentication.
// This uses the default serializer to marshal and unmarshal messages.
type AuthnManageClient struct {
	// ID of the authn service
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (mngAuthn *AuthnManageClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}
	data, err := mngAuthn.hc.PubAction(
		mngAuthn.serviceID, auth.ManageAuthnCapability, action, msg)

	err = mngAuthn.hc.ParseResponse(data, err, resp)

	return err
}

// AddDevice adds an IoT device and generates an authentication token
func (mngAuthn *AuthnManageClient) AddDevice(
	deviceID string, displayName string, pubKey string, tokenValidity time.Duration) (string, error) {

	slog.Info("AddDevice", "deviceID", deviceID)
	req := auth.AddDeviceReq{
		DeviceID:      deviceID,
		DisplayName:   displayName,
		PubKey:        pubKey,
		TokenValidity: tokenValidity,
	}
	resp := auth.AddDeviceResp{}
	err := mngAuthn.pubReq(auth.AddDeviceAction, &req, &resp)
	return resp.Token, err
}

// AddService adds a service.
func (mngAuthn *AuthnManageClient) AddService(
	serviceID string, displayName string, pubKey string, tokenValidity time.Duration) (string, error) {

	slog.Info("AddService", "serviceID", serviceID)
	req := auth.AddServiceReq{
		ServiceID:     serviceID,
		DisplayName:   displayName,
		PubKey:        pubKey,
		TokenValidity: tokenValidity,
	}
	resp := auth.AddServiceResp{}
	err := mngAuthn.pubReq(auth.AddServiceAction, &req, &resp)
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
func (mngAuthn *AuthnManageClient) AddUser(userID string, displayName string, password string, pubKey string) (string, error) {
	slog.Info("AddUser", "userID", userID)
	req := auth.AddUserReq{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
	}
	resp := auth.AddUserResp{}
	err := mngAuthn.pubReq(auth.AddUserAction, &req, &resp)
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (mngAuthn *AuthnManageClient) GetCount() (n int, err error) {
	resp := auth.GetCountResp{}
	err = mngAuthn.pubReq(auth.GetCountAction, nil, &resp)
	return resp.N, err
}

// GetAuthClientList provides a list of clients to apply to the message server
//func (mngAuthn *AuthnManageClient) GetAuthClientList() []msgserver.AuthClient {
//	//FIXME
//}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (mngAuthn *AuthnManageClient) GetProfile(clientID string) (profile auth.ClientProfile, err error) {
	req := auth.GetProfileReq{
		ClientID: clientID,
	}
	resp := auth.GetProfileResp{}
	err = mngAuthn.pubReq(auth.GetProfileAction, &req, &resp)
	return resp.Profile, err
}

// GetProfiles provide a list of known clients and their info.
// The caller must be an administrator or service.
func (mngAuthn *AuthnManageClient) GetProfiles() (profiles []auth.ClientProfile, err error) {
	resp := auth.GetProfilesResp{}
	err = mngAuthn.pubReq(auth.GetProfilesAction, nil, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (mngAuthn *AuthnManageClient) RemoveUser(clientID string) error {
	req := auth.RemoveClientReq{
		ClientID: clientID,
	}
	err := mngAuthn.pubReq(auth.RemoveClientAction, &req, nil)
	return err
}

// UpdateClient updates a client's profile
func (mngAuthn *AuthnManageClient) UpdateUser(clientID string, prof auth.ClientProfile) error {
	req := &auth.UpdateClientReq{
		ClientID: clientID,
		Profile:  prof,
	}
	err := mngAuthn.pubReq(auth.UpdateClientAction, req, nil)
	return err
}

// NewAuthnManageClient returns an authn management client for the given hubclient connection
func NewAuthnManageClient(hc hubclient.IHubClient) auth.IAuthnManage {
	bindingID := auth.AuthServiceName

	cl := AuthnManageClient{
		hc:        hc,
		serviceID: bindingID,
	}
	return &cl
}
