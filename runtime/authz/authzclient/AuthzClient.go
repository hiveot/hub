package authzclient

import (
	"github.com/hiveot/hub/runtime/api"
)

type AuthzClient struct {
	mt api.IMessageTransport
}

func (svc *AuthzClient) GetClientRole(clientID string) (string, error) {
	args := api.GetClientRoleArgs{ClientID: clientID}
	resp := api.GetClientRoleResp{}
	err := svc.mt(api.AuthzThingID, api.GetClientRoleMethod, &args, &resp)
	return resp.Role, err
}

func (svc *AuthzClient) SetClientRole(clientID string, role string) error {
	args := api.SetClientRoleArgs{ClientID: clientID, Role: role}
	err := svc.mt(api.AuthzThingID, api.SetClientRoleMethod, &args, nil)
	return err
}

// NewAuthzClient creates a new instance of the authorization client
func NewAuthzClient(mt api.IMessageTransport) *AuthzClient {
	cl := AuthzClient{mt: mt}
	return &cl
}
