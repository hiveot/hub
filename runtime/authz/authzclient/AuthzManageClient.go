package authzclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// AuthzManageClient marshals and unmarshals request to the authz management service
type AuthzManageClient struct {
	hc       hubclient.IHubClient
	dThingID string
}

func (svc *AuthzManageClient) GetClientRole(clientID string) (string, error) {
	args := api.GetClientRoleArgs{ClientID: clientID}
	resp := api.GetClientRoleResp{}
	err := svc.hc.Rpc(svc.dThingID, api.GetClientRoleMethod, &args, &resp)
	return resp.Role, err
}

func (svc *AuthzManageClient) SetClientRole(clientID string, role string) error {
	args := api.SetClientRoleArgs{ClientID: clientID, Role: role}
	err := svc.hc.Rpc(svc.dThingID, api.SetClientRoleMethod, &args, nil)
	return err
}

// NewAuthzManageClient creates a new instance of the authorization management client
func NewAuthzManageClient(hc hubclient.IHubClient) *AuthzManageClient {
	dThingID := things.MakeDigiTwinThingID(api.AuthzAgentID, api.AuthzManageServiceID)
	cl := AuthzManageClient{hc: hc, dThingID: dThingID}
	return &cl
}
