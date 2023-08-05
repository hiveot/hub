package authnclient

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
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
func (mngAuthn *AuthnManageClient) pubReq(action string, req interface{}) ([]byte, error) {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}
	return mngAuthn.hc.PubAction(
		mngAuthn.serviceID, authn.ManageAuthnCapability, action, msg)
}

// AddDevice adds an IoT device and generates an authentication token
func (mngAuthn *AuthnManageClient) AddDevice(deviceID string, displayName string, pubKey string, validitySec int) (string, error) {
	slog.Info("AddDevice", "deviceID", deviceID)
	req := &authn.AddDeviceReq{
		DeviceID:    deviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
		ValiditySec: validitySec,
	}
	data, err := mngAuthn.pubReq(authn.AddDeviceAction, req)
	resp := authn.AddDeviceResp{}
	err = mngAuthn.hc.ParseResponse(data, err, &resp)
	return resp.Token, err
}

// AddService adds a service.
func (mngAuthn *AuthnManageClient) AddService(serviceID string, displayName string, pubKey string, validitySec int) (string, error) {
	slog.Info("AddService", "serviceID", serviceID)
	req := &authn.AddServiceReq{
		ServiceID:   serviceID,
		DisplayName: displayName,
		PubKey:      pubKey,
		ValiditySec: validitySec,
	}
	data, err := mngAuthn.pubReq(authn.AddServiceAction, req)
	resp := authn.AddServiceResp{}
	err = mngAuthn.hc.ParseResponse(data, err, &resp)
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
	req := &authn.AddUserReq{
		UserID:      userID,
		DisplayName: displayName,
		Password:    password,
		PubKey:      pubKey,
	}
	data, err := mngAuthn.pubReq(authn.AddUserAction, req)
	resp := authn.AddUserResp{}
	err = mngAuthn.hc.ParseResponse(data, err, &resp)
	return resp.Token, err
}

// GetCount returns the number of clients in the store
func (mngAuthn *AuthnManageClient) GetCount() (n int, err error) {
	data, err := mngAuthn.pubReq(authn.GetCountAction, nil)
	resp := &authn.GetCountResp{}
	err = mngAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		n = resp.N
	}
	return n, err
}

// GetClientProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (mngAuthn *AuthnManageClient) GetClientProfile(clientID string) (profile authn.ClientProfile, err error) {
	req := authn.GetClientProfileReq{
		ClientID: clientID,
	}
	data, err := mngAuthn.pubReq(authn.GetClientProfileAction, req)
	resp := &authn.GetProfileResp{}
	err = mngAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		profile = resp.Profile
	}
	return profile, err
}

// ListClients provide a list of known clients and their info.
// The caller must be an administrator or service.
func (mngAuthn *AuthnManageClient) ListClients() (profiles []authn.ClientProfile, err error) {
	data, err := mngAuthn.pubReq(authn.ListClientsAction, nil)
	resp := &authn.ListClientsResp{}
	err = mngAuthn.hc.ParseResponse(data, err, resp)
	if err == nil {
		profiles = resp.Profiles
	}
	return profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (mngAuthn *AuthnManageClient) RemoveClient(clientID string) error {
	req := &authn.RemoveClientReq{
		ClientID: clientID,
	}
	data, err := mngAuthn.pubReq(authn.RemoveClientAction, req)
	err = mngAuthn.hc.ParseResponse(data, err, nil)
	return err
}

// UpdateClient updates a client's profile
func (mngAuthn *AuthnManageClient) UpdateClient(clientID string, prof authn.ClientProfile) error {
	req := &authn.UpdateClientReq{
		ClientID: clientID,
		Profile:  prof,
	}
	data, err := mngAuthn.pubReq(authn.UpdateClientAction, req)
	err = mngAuthn.hc.ParseResponse(data, err, nil)
	return err
}

// NewAuthnManageClient returns an authn management client for the given hubclient connection
func NewAuthnManageClient(hc hubclient.IHubClient) authn.IAuthnManage {
	bindingID := authn.AuthnServiceName

	cl := AuthnManageClient{
		hc:        hc,
		serviceID: bindingID,
	}
	return &cl
}
