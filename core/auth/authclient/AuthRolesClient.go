package authclient

import (
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// AuthRolesClient is a marshaller for messaging to manage custom roles
// This uses the default serializer to marshal and unmarshal messages.
type AuthRolesClient struct {
	// ID of the auth service instance
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an action service request to the auth service
func (cl *AuthRolesClient) pubReq(action string, msg []byte, resp interface{}) error {
	data, err := cl.hc.PubServiceRPC(
		cl.serviceID, auth2.AuthRolesCapability, action, msg)
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

	req := auth2.CreateRoleArgs{
		Role: role,
	}
	msg, _ := ser.Marshal(req)
	err := cl.pubReq(auth2.CreateRoleReq, msg, nil)
	return err
}

// DeleteRole deletes a custom role
func (cl *AuthRolesClient) DeleteRole(role string) error {

	req := auth2.DeleteRoleArgs{
		Role: role,
	}
	msg, _ := ser.Marshal(req)
	err := cl.pubReq(auth2.DeleteRoleReq, msg, nil)
	return err
}

// NewAuthRolesClient creates a new client for managing roles
//
//	hc is the hub client connection to use
func NewAuthRolesClient(hc hubclient.IHubClient) auth2.IAuthManageRoles {
	cl := &AuthRolesClient{
		serviceID: auth2.AuthServiceName,
		hc:        hc,
	}
	return cl

}
