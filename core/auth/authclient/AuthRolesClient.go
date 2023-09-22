package authclient

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// AuthRolesClient is a marshaller for messaging to manage roles
// This uses the default serializer to marshal and unmarshal messages.
type AuthRolesClient struct {
	// ID of the authz service
	hc hubclient.IHubClient
}

// helper for publishing an action service request to the auth service
func (cl *AuthRolesClient) pubReq(action string, msg []byte, resp interface{}) error {
	data, err := cl.hc.PubServiceRPC(
		auth.AuthServiceName, auth.AuthRolesCapability, action, msg)
	if err != nil {
		return err
	}
	if data.ErrorReply != nil {
		return data.ErrorReply
	}
	err = cl.hc.ParseResponse(data.Payload, resp)
	return err
}

// CreateRole creates a new custom role
func (cl *AuthRolesClient) CreateRole(role string) error {

	req := auth.CreateRoleReq{
		Role: role,
	}
	msg, _ := ser.Marshal(req)
	err := cl.pubReq(auth.CreateRoleAction, msg, nil)
	return err
}

// DeleteRole deletes a custom role
func (cl *AuthRolesClient) DeleteRole(role string) error {

	req := auth.DeleteRoleReq{
		Role: role,
	}
	msg, _ := ser.Marshal(req)
	err := cl.pubReq(auth.DeleteRoleAction, msg, nil)
	return err
}

// SetRole updates the role for a client
func (cl *AuthRolesClient) SetRole(clientID string, userRole string) error {

	req := auth.SetRoleReq{
		ClientID: clientID,
		Role:     userRole,
	}
	msg, _ := ser.Marshal(req)
	err := cl.pubReq(auth.SetRoleAction, msg, nil)
	return err
}

// NewAuthRolesClient creates a new client for managing roles
func NewAuthRolesClient(hc hubclient.IHubClient) auth.IAuthManageRoles {
	cl := &AuthRolesClient{
		hc: hc,
	}
	return cl

}
