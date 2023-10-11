package capnpclient

import (
	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"context"
	"github.com/hiveot/hub/pkg/provisioning"

	"github.com/hiveot/hub/api/go/hubapi"
)

// ProvisioningCapnpClient provides a POGS wrapper around the generated provisioning capnp client
// This implements the IProvisioning interface
type ProvisioningCapnpClient struct {
	connection *rpc.Conn              // connection to the capnp server
	capability hubapi.CapProvisioning // capnp client
}

// CapManageProvisioning provides the capability to manage provisioning requests
func (cl *ProvisioningCapnpClient) CapManageProvisioning(
	ctx context.Context, clientID string) (provisioning.IManageProvisioning, error) {

	getCap, release := cl.capability.CapManageProvisioning(ctx,
		func(params hubapi.CapProvisioning_capManageProvisioning_Params) error {
			err2 := params.SetClientID(clientID)
			return err2
		})
	defer release()
	capability := getCap.Cap()
	newCap := NewManageProvisioningCapnpClient(capability.AddRef())
	return newCap, nil
}

// CapRequestProvisioning provides the capability to provision IoT devices
func (cl *ProvisioningCapnpClient) CapRequestProvisioning(
	ctx context.Context, clientID string) (provisioning.IRequestProvisioning, error) {

	getCap, release := cl.capability.CapRequestProvisioning(ctx,
		func(params hubapi.CapProvisioning_capRequestProvisioning_Params) error {
			err2 := params.SetClientID(clientID)
			return err2
		})
	defer release()
	capability := getCap.Cap()
	newCap := NewRequestProvisioningCapnpClient(capability.AddRef())
	return newCap, nil
}

// CapRefreshProvisioning provides the capability for IoT devices to refresh
func (cl *ProvisioningCapnpClient) CapRefreshProvisioning(
	ctx context.Context, clientID string) (provisioning.IRefreshProvisioning, error) {

	getCap, release := cl.capability.CapRefreshProvisioning(ctx,
		func(params hubapi.CapProvisioning_capRefreshProvisioning_Params) error {
			err2 := params.SetClientID(clientID)
			return err2
		})
	defer release()
	capability := getCap.Cap()
	newCap := NewRefreshProvisioningCapnpClient(capability.AddRef())
	return newCap, nil
}

// Release the client capability
func (cl *ProvisioningCapnpClient) Release() {
	cl.capability.Release()
	if cl.connection != nil {
		_ = cl.connection.Close()
	}
}

// NewProvisioningCapnpClient returns a provisioning service client using the capnp protocol
func NewProvisioningCapnpClient(client capnp.Client) *ProvisioningCapnpClient {
	capability := hubapi.CapProvisioning(client)

	cl := &ProvisioningCapnpClient{
		connection: nil,
		capability: capability,
	}
	return cl
}
