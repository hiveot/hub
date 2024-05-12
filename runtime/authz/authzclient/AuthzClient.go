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
	stat, err := svc.hc.Rpc(nil, api.AuthzThingID, api.GetClientRoleMethod, &args, &resp)
	_ = stat
	return resp.Role, err
}

func (svc *AuthzClient) SetClientRole(clientID string, role string) error {
	args := api.SetClientRoleArgs{ClientID: clientID, Role: role}
	stat, err := svc.hc.Rpc(nil, api.AuthzThingID, api.SetClientRoleMethod, &args, nil)
	_ = stat
	return err
}

// NewAuthzClient creates a new instance of the authorization client
func NewAuthzClient(hc hubclient.IHubClient) *AuthzClient {
	cl := AuthzClient{hc: hc}
	return &cl
}
