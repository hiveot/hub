package idprovclient

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/idprov/idprovapi"
	"github.com/hiveot/hub/lib/hubclient"
)

// ManageIdProvClient is a hiveot client for communicating with the provisioning
// service using the message bus.
// This requires admin permissions.
type ManageIdProvClient struct {
	hc hubclient.IHubClient
	// agentID of the service
	serviceID string
	// capabilityID of this capability
	capID string
}

// ApproveRequest approves a pending provisioning request
func (cl *ManageIdProvClient) ApproveRequest(ClientID string, clientType string) error {
	args := idprovapi.ApproveRequestArgs{
		ClientID:   ClientID,
		ClientType: clientType,
	}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID,
		idprovapi.ApproveRequestMethod, &args, nil)

	return err
}

// GetRequests returns requests
// Expired requests are not included.
func (cl *ManageIdProvClient) GetRequests(
	pending, approved, rejected bool) ([]idprovapi.ProvisionStatus, error) {
	args := idprovapi.GetRequestsArgs{
		Pending:  pending,
		Approved: approved,
		Rejected: rejected,
	}
	resp := idprovapi.GetRequestsResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID,
		idprovapi.GetRequestsMethod, &args, &resp)

	return resp.Requests, err
}

// PreApproveDevices uploads a list of pre-approved devices ID, MAC and PubKey
func (cl *ManageIdProvClient) PreApproveDevices(
	approvals []idprovapi.PreApprovedClient) error {

	args := idprovapi.PreApproveClientsArgs{
		Approvals: approvals,
	}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID,
		idprovapi.PreApproveClientsMethod, &args, nil)

	return err
}

// RejectRequest rejects a pending provisioning request
func (cl *ManageIdProvClient) RejectRequest(clientID string) error {
	args := idprovapi.RejectRequestArgs{ClientID: clientID}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID,
		idprovapi.RejectRequestMethod, &args, nil)

	return err
}
func (cl *ManageIdProvClient) SubmitRequest(
	clientID string, pubKey string, mac string) (
	status *idprovapi.ProvisionStatus, token string, err error) {

	args := idprovapi.SubmitRequestArgs{
		ClientID:   clientID,
		ClientType: authapi.ClientTypeDevice,
		PubKey:     pubKey,
		MAC:        mac,
	}
	resp := idprovapi.ProvisionRequestResp{}
	_, err = cl.hc.PubRPCRequest(cl.serviceID, cl.capID,
		idprovapi.SubmitRequestMethod, &args, &resp)

	return &resp.Status, resp.Token, err
}

func NewIdProvManageClient(hc hubclient.IHubClient) *ManageIdProvClient {
	cl := &ManageIdProvClient{
		hc: hc,
		//
		serviceID: idprovapi.ServiceName,
		capID:     idprovapi.ManageProvisioningCap,
	}
	return cl
}
