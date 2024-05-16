package idprovclient

import (
	"errors"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/services/idprov/idprovapi"
)

// ManageIdProvClient is a hiveot client for communicating with the provisioning
// service using the message bus.
// This requires admin permissions.
type ManageIdProvClient struct {
	hc hubclient.IHubClient
	// agentID of the service capability (Thing)
	agentID string
	// thingID of the capability (digitwin version with agent prefix)
	thingID string
}

// Invoke the IDProv method
// This returns an error if anything goes wrong: not delivered, delivery incomplete or processing error
func (cl *ManageIdProvClient) call(method string, args interface{}, resp interface{}) error {
	stat, err := cl.hc.Rpc(nil, cl.thingID, method, args, resp)
	if err == nil {
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if stat.Status != api.DeliveryCompleted {
			err = errors.New("No error but delivery not completed")
		}
	}
	return err
}

// ApproveRequest approves a pending provisioning request
func (cl *ManageIdProvClient) ApproveRequest(ClientID string, clientType string) error {
	args := idprovapi.ApproveRequestArgs{
		ClientID:   ClientID,
		ClientType: clientType,
	}
	err := cl.call(idprovapi.ApproveRequestMethod, &args, nil)
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
	err := cl.call(idprovapi.GetRequestsMethod, &args, &resp)
	return resp.Requests, err
}

// PreApproveDevices uploads a list of pre-approved devices ID, MAC and PubKey
func (cl *ManageIdProvClient) PreApproveDevices(
	approvals []idprovapi.PreApprovedClient) error {

	args := idprovapi.PreApproveClientsArgs{
		Approvals: approvals,
	}
	err := cl.call(idprovapi.PreApproveClientsMethod, &args, nil)
	return err
}

// RejectRequest rejects a pending provisioning request
func (cl *ManageIdProvClient) RejectRequest(clientID string) error {
	args := idprovapi.RejectRequestArgs{ClientID: clientID}
	err := cl.call(idprovapi.RejectRequestMethod, &args, nil)
	return err
}
func (cl *ManageIdProvClient) SubmitRequest(
	clientID string, pubKey string, mac string) (
	status *idprovapi.ProvisionStatus, token string, err error) {

	args := idprovapi.SubmitRequestArgs{
		ClientID:   clientID,
		ClientType: api.ClientTypeAgent,
		PubKey:     pubKey,
		MAC:        mac,
	}
	resp := idprovapi.ProvisionRequestResp{}
	err = cl.call(idprovapi.RejectRequestMethod, &args, &resp)
	return &resp.Status, resp.Token, err
}

func NewIdProvManageClient(hc hubclient.IHubClient) *ManageIdProvClient {
	cl := &ManageIdProvClient{
		hc: hc,
		//
		agentID: idprovapi.AgentID,
		thingID: idprovapi.ManageProvisioningThingID,
	}
	return cl
}
