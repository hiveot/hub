package idprovclient

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	"github.com/hiveot/hub/wot/tdd"
)

// ManageIdProvClient is a hiveot client for communicating with the provisioning
// service using the message bus.
// This requires admin permissions.
type ManageIdProvClient struct {
	hc clients.IConsumer
	// thingID digital twin service ID of this capability (digitwin version with agent prefix)
	dThingID string
}

// _send the IDProv method
// This returns an error if anything goes wrong: not delivered, delivery incomplete or processing error
func (cl *ManageIdProvClient) call(method string, args interface{}, resp interface{}) error {
	err := cl.hc.Rpc(cl.dThingID, method, args, resp)
	return err
}

// ApproveRequest approves a pending provisioning request
func (cl *ManageIdProvClient) ApproveRequest(ClientID string, clientType authn.ClientType) error {
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
		ClientType: authn.ClientTypeAgent,
		PubKey:     pubKey,
		MAC:        mac,
	}
	resp := idprovapi.ProvisionRequestResp{}
	err = cl.call(idprovapi.RejectRequestMethod, &args, &resp)
	return &resp.Status, resp.Token, err
}

func NewIdProvManageClient(hc clients.IConsumer) *ManageIdProvClient {
	agentID := idprovapi.AgentID
	cl := &ManageIdProvClient{
		hc:       hc,
		dThingID: tdd.MakeDigiTwinThingID(agentID, idprovapi.ManageServiceID),
	}
	return cl
}
