package client

import (
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// ManageAuthn is a client for the authn service for administrators and services that manage authentication
// This uses the default serializer to marshal and unmarshal messages.
type ManageAuthn struct {
	// ID of the authn service
	serviceID string
	hc        hub.IHubClient
}

// helper for publishing an action request to the authz service
func (mngAuthn *ManageAuthn) pubReq(action string, msg []byte) ([]byte, error) {
	return mngAuthn.hc.PubAction(mngAuthn.serviceID, "", action, msg)
}

// AddUser adds a user.
// The caller must be an administrator or service.
// If the userID already exists then an error is returned
//
//	userID is the login ID of the user, typically their email
//	name of the user for presentation
//	password the user can login with if their token has expired.
func (mngAuthn *ManageAuthn) AddUser(userID string, name string, password string) error {
	req := authn.AddUserReq{
		UserID:   userID,
		Name:     name,
		Password: password,
	}
	msg, _ := ser.Marshal(req)
	data, err := mngAuthn.pubReq(authn.AddUserAction, msg)
	err = hubclient.ParseResponse(data, err, nil)
	return err
}

// GetProfile returns a client's profile
// Users can only get their own profile.
// Managers can get other clients profiles.
func (mngAuthn *ManageAuthn) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	req := authn.GetProfileReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := mngAuthn.pubReq(authn.GetProfileAction, msg)
	resp := &authn.GetProfileResp{}
	err = hubclient.ParseResponse(data, err, resp)
	if err == nil {
		profile = resp.Profile
	}
	return profile, err
}

// ListClients provide a list of known clients and their info.
// The caller must be an administrator or service.
func (mngAuthn *ManageAuthn) ListClients() (profiles []authn.ClientProfile, err error) {
	data, err := mngAuthn.pubReq(authn.ListClientsAction, nil)
	resp := &authn.ListClientsResp{}
	err = hubclient.ParseResponse(data, err, resp)
	if err == nil {
		profiles = resp.Profiles
	}
	return profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (mngAuthn *ManageAuthn) RemoveClient(clientID string) error {
	req := authn.RemoveClientReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := mngAuthn.pubReq(authn.RemoveClientAction, msg)
	err = hubclient.ParseResponse(data, err, nil)
	return err
}

// NewManageAuthn returns an authn management client for the given hubclient connection
func NewManageAuthn(hc hub.IHubClient) authn.IManageAuthn {
	cl := ManageAuthn{
		hc:        hc,
		serviceID: "authn",
	}
	return &cl
}
