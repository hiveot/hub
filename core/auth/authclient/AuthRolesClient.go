package authclient

import (
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
)

// AuthRolesClient is a marshaller for messaging to manage custom roles
// This uses the default serializer to marshal and unmarshal messages.
type AuthRolesClient struct {
	// ID of the authn service agent
	agentID string
	// capability to invoke
	capID string
	hc    hubclient.IHubClient
}

// CreateRole creates a new custom role
func (cl *AuthRolesClient) CreateRole(role string) error {

	req := auth.CreateRoleArgs{
		Role: role,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.CreateRoleReq, &req, nil)
	return err
}

// DeleteRole deletes a custom role
func (cl *AuthRolesClient) DeleteRole(role string) error {

	req := auth.DeleteRoleArgs{
		Role: role,
	}
	_, err := cl.hc.PubRPCRequest(
		cl.agentID, cl.capID, auth.DeleteRoleReq, &req, nil)
	return err
}

// NewAuthRolesClient creates a new client for managing roles
//
//	hc is the hub client connection to use
func NewAuthRolesClient(hc hubclient.IHubClient) auth.IAuthManageRoles {
	cl := &AuthRolesClient{
		agentID: auth.AuthServiceName,
		capID:   auth.AuthManageRolesCapability,
		hc:      hc,
	}
	return cl

}
