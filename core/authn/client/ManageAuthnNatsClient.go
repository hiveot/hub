package client

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
)

// ManageAuthn is a client for the authn service for administrators and services that manage authentication
// This uses the default serializer to marshal and unmarshal messages.
type ManageAuthn struct {
	// ID of the authn service
	serviceID string
	hc        *hubconn.HubConnNats
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
	_, err := mngAuthn.pubReq(authn.AddUserAction, msg)
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
	if err != nil {
		return profile, err
	}
	resp := &authn.GetProfileResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Profile, err
}

// ListClients provide a list of known clients and their info.
// The caller must be an administrator or service.
func (mngAuthn *ManageAuthn) ListClients() (profiles []authn.ClientProfile, err error) {
	data, err := mngAuthn.pubReq(authn.ListClientsAction, nil)
	if err != nil {
		return nil, err
	}
	resp := &authn.ListClientsResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Profiles, err
}

// RemoveClient removes a client and disables authentication
// Existing tokens are immediately expired (tbd)
func (mngAuthn *ManageAuthn) RemoveClient(clientID string) error {
	req := authn.RemoveClientReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	_, err := mngAuthn.pubReq(authn.RemoveClientAction, msg)
	return err
}

// ResetPassword reset a user's login password
func (mngAuthn *ManageAuthn) ResetPassword(clientID string, newPassword string) error {
	req := authn.ResetPasswordReq{
		ClientID: clientID,
		Password: newPassword,
	}
	msg, _ := ser.Marshal(req)
	_, err := mngAuthn.pubReq(authn.ResetPasswordAction, msg)
	return err
}

// NewManageAuthn returns an authn management client for the given hubclient connection
func NewManageAuthn(hc *hubconn.HubConnNats) authn.IClientAuthn {
	cl := ClientAuthn{
		hc:        hc,
		serviceID: "authn",
	}
	return &cl
}
