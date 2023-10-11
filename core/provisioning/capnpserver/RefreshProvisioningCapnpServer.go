package capnpserver

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capserializer"

	"github.com/hiveot/hub/api/go/hubapi"
)

// RefreshProvisioningCapnpServer provides the capnproto RPC server to Refresh device provisioning
type RefreshProvisioningCapnpServer struct {
	pogosrv provisioning.IRefreshProvisioning
}

func (capsrv *RefreshProvisioningCapnpServer) RefreshDeviceCert(
	ctx context.Context, call hubapi.CapRefreshProvisioning_refreshDeviceCert) error {

	args := call.Args()
	//deviceID, _ := args.DeviceID()
	certPEM, _ := args.CertPEM()
	status, err := capsrv.pogosrv.RefreshDeviceCert(ctx, certPEM)
	if err == nil {
		res, _ := call.AllocResults()
		provStatusCapnp := capserializer.MarshalProvStatus(status)
		err = res.SetStatus(provStatusCapnp)
	}

	return err
}
