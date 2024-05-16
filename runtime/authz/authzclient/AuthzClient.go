package authzclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
)

// AuthzClient marshals and unmarshals request to the authz service
type AuthzClient struct {
	hc hubclient.IHubClient
}

func (svc *AuthzClient) GetClientRole(clientID string) (string, error) {
	args := api.GetClientRoleArgs{ClientID: clientID}
	resp := api.GetClientRoleResp{}
	err := svc.hc.Rpc(api.AuthzThingID, api.GetClientRoleMethod, &args, &resp)
	return resp.Role, err
}

func (svc *AuthzClient) SetClientRole(clientID string, role string) error {
	args := api.SetClientRoleArgs{ClientID: clientID, Role: role}
	err := svc.hc.Rpc(api.AuthzThingID, api.SetClientRoleMethod, &args, nil)
	return err
}

// NewAuthzClient creates a new instance of the authorization client
func NewAuthzClient(hc hubclient.IHubClient) *AuthzClient {
	cl := AuthzClient{hc: hc}
	return &cl
}
