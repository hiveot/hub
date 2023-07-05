package client

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
)

// ClientAuthn is a marshaller for messaging with the authn service using the hub client connection.
// This is intended for clients to authenticate themselves and refresh auth tokens.
// This uses the default serializer to marshal and unmarshal messages.
type ClientAuthn struct {
	// ID of the authn service
	serviceID string
	hc        *hubconn.HubConnNats
}

// helper for publishing an action request to the authz service
func (clientAuthn *ClientAuthn) pubReq(action string, msg []byte) ([]byte, error) {
	return clientAuthn.hc.PubAction(clientAuthn.serviceID, "", action, msg)
}

// Login to obtain an auth token
func (clientAuthn *ClientAuthn) Login(clientID string, password string) (authToken string, err error) {
	req := authn.LoginReq{
		ClientID: clientID,
		Password: password,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.LoginAction, msg)
	if err != nil {
		return authToken, err
	}
	resp := &authn.LoginResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.AuthToken, err
}

// Logout logs out the user and invalidates the authentication token
func (clientAuthn *ClientAuthn) Logout(authToken string) error {
	req := authn.LogoutReq{
		AuthToken: authToken,
	}
	msg, _ := ser.Marshal(req)
	_, err := clientAuthn.pubReq(authn.LogoutAction, msg)
	return err
}

// Refresh a short-lived authentication token.
func (clientAuthn *ClientAuthn) Refresh(clientID string, oldToken string) (authToken string, err error) {
	req := authn.RefreshReq{
		ClientID: clientID,
		OldToken: oldToken,
	}
	msg, _ := ser.Marshal(req)
	data, err := clientAuthn.pubReq(authn.RefreshAction, msg)
	if err != nil {
		return authToken, err
	}
	resp := &authn.RefreshResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.AuthToken, err
}

// UpdateName updates a user's display name
func (clientAuthn *ClientAuthn) UpdateName(clientID string, newName string) error {
	req := authn.UpdateNameReq{
		ClientID: clientID,
		NewName:  newName,
	}
	msg, _ := ser.Marshal(req)
	_, err := clientAuthn.pubReq(authn.UpdateNameAction, msg)
	return err
}

// UpdatePassword changes the client password
// Login or Refresh must be called successfully first.
func (clientAuthn *ClientAuthn) UpdatePassword(clientID string, newPassword string) error {
	req := authn.UpdatePasswordReq{
		ClientID:    clientID,
		NewPassword: newPassword,
	}
	msg, _ := ser.Marshal(req)
	_, err := clientAuthn.pubReq(authn.UpdatePasswordAction, msg)
	return err
}

// NewClientAuthn returns an authn client for the given hubclient connection
func NewClientAuthn(hc *hubconn.HubConnNats) authn.IClientAuthn {
	cl := ClientAuthn{
		hc:        hc,
		serviceID: "authn",
	}
	return &cl
}
