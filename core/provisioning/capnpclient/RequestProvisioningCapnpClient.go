package capnpclient

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capserializer"

	"github.com/hiveot/hub/api/go/hubapi"
)

// RequestProvisioningCapnpClient provides the POGS interface with the capability to send provisioning requests
type RequestProvisioningCapnpClient struct {
	// The capnp client
	capability hubapi.CapRequestProvisioning
}

// Release the client and release its resources
func (cl *RequestProvisioningCapnpClient) Release() {
	cl.capability.Release()
}

// SubmitProvisioningRequest passes the provisioning request to the server via the capnp protocol
func (cl *RequestProvisioningCapnpClient) SubmitProvisioningRequest(
	ctx context.Context, deviceID string, md5Secret string, pubKeyPEM string) (
	provStatus provisioning.ProvisionStatus, err error) {

	method, release := cl.capability.SubmitProvisioningRequest(ctx,
		func(params hubapi.CapRequestProvisioning_submitProvisioningRequest_Params) error {
			err2 := params.SetDeviceID(deviceID)
			_ = params.SetMd5Secret(md5Secret)
			_ = params.SetPubKeyPEM(pubKeyPEM)
			return err2
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		provStatusCapnp, err2 := resp.Status()
		err = err2
		provStatus = capserializer.UnmarshalProvStatus(provStatusCapnp)
	}
	return provStatus, err
}

// NewRequestProvisioningCapnpClient returns an instance of the POGS wrapper around the capnp api
func NewRequestProvisioningCapnpClient(cap hubapi.CapRequestProvisioning) *RequestProvisioningCapnpClient {
	cl := &RequestProvisioningCapnpClient{capability: cap}
	return cl
}
