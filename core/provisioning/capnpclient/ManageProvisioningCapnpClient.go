package capnpclient

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capserializer"

	"github.com/hiveot/hub/api/go/hubapi"
)

// ManageProvisioningCapnpClient provides the POGS interface with the capability to manage provisioning requests
type ManageProvisioningCapnpClient struct {
	// The capnp client
	capability hubapi.CapManageProvisioning
}

// Release the client and release its resources
func (cl *ManageProvisioningCapnpClient) Release() {
	cl.capability.Release()
}

// AddOOBSecrets adds a list of OOB secrets for automated provisioning
func (cl *ManageProvisioningCapnpClient) AddOOBSecrets(
	ctx context.Context, oobSecrets []provisioning.OOBSecret) error {

	method, release := cl.capability.AddOOBSecrets(ctx,
		func(params hubapi.CapManageProvisioning_addOOBSecrets_Params) error {
			secretsListCapnp := capserializer.MarshalOobSecrets(oobSecrets)
			err2 := params.SetOobSecrets(secretsListCapnp)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

// ApproveRequest approves a pending request for the given device ID
func (cl *ManageProvisioningCapnpClient) ApproveRequest(
	ctx context.Context, deviceID string) error {

	method, release := cl.capability.ApproveRequest(ctx,
		func(params hubapi.CapManageProvisioning_approveRequest_Params) error {
			err2 := params.SetDeviceID(deviceID)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

// GetApprovedRequests returns a list of approved requests
func (cl *ManageProvisioningCapnpClient) GetApprovedRequests(
	ctx context.Context) ([]provisioning.ProvisionStatus, error) {

	statusList := make([]provisioning.ProvisionStatus, 0)
	method, release := cl.capability.GetApprovedRequests(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		provStatusListCapnp, _ := resp.Requests()
		statusList = capserializer.UnmarshalProvStatusList(provStatusListCapnp)
	}
	return statusList, err
}

// GetPendingRequests returns a list of pending requests
func (cl *ManageProvisioningCapnpClient) GetPendingRequests(
	ctx context.Context) ([]provisioning.ProvisionStatus, error) {

	statusList := make([]provisioning.ProvisionStatus, 0)
	method, release := cl.capability.GetPendingRequests(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		provStatusListCapnp, _ := resp.Requests()
		statusList = capserializer.UnmarshalProvStatusList(provStatusListCapnp)
	}
	return statusList, err
}

// NewManageProvisioningCapnpClient returns an instance of the POGS wrapper around the capnp api
func NewManageProvisioningCapnpClient(cap hubapi.CapManageProvisioning) *ManageProvisioningCapnpClient {
	cl := &ManageProvisioningCapnpClient{capability: cap}
	return cl
}
