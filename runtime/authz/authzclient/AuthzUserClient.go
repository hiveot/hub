package authzclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// AuthzUserClient marshals and unmarshals request to the authz management service
type AuthzUserClient struct {
	hc       hubclient.IHubClient
	dThingID string
}

// SetPermissions sets roles allowed/denied to use the service offered by this agent.
// This is intended for agents offering their service to consumers.
//
// These permissions are default recommendations made by the service provider. The
// authz service can choose to override these defaults with another configuration.
//
//	agentID is the agent offering the service
//	serviceID is the agent's service/capability ID whose roles to set. (not the digitwin ID)
//	roles is a list of consumer roles allowed to use the service.
func (cl *AuthzUserClient) SetPermissions(p api.ThingPermissions) error {

	err := cl.hc.Rpc(cl.dThingID, api.SetPermissionsMethod, &p, nil)
	return err
}

// NewAuthzUserClient creates a new instance of the authorization user client
func NewAuthzUserClient(hc hubclient.IHubClient) *AuthzUserClient {
	dThingID := things.MakeDigiTwinThingID(api.AuthzAgentID, api.AuthzUserServiceID)
	cl := AuthzUserClient{hc: hc, dThingID: dThingID}
	return &cl
}
