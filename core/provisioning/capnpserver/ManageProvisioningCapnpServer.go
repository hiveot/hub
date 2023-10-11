package capnpserver

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capserializer"

	"github.com/hiveot/hub/api/go/hubapi"
)

// ManageProvisioningCapnpServer provides the capnproto RPC server for IOT device provisioning.
// This implements the capnproto generated interface CapManageProvisioning_Server
type ManageProvisioningCapnpServer struct {
	pogosrv provisioning.IManageProvisioning
}

func (capsrv *ManageProvisioningCapnpServer) AddOOBSecrets(
	ctx context.Context, call hubapi.CapManageProvisioning_addOOBSecrets) error {

	args := call.Args()
	secretsCapnp, _ := args.OobSecrets()
	secretsPOGS := capserializer.UnmarshalOobSecrets(secretsCapnp)
	err := capsrv.pogosrv.AddOOBSecrets(ctx, secretsPOGS)
	return err
}

func (capsrv *ManageProvisioningCapnpServer) ApproveRequest(
	ctx context.Context, call hubapi.CapManageProvisioning_approveRequest) error {

	args := call.Args()
	deviceID, _ := args.DeviceID()
	err := capsrv.pogosrv.ApproveRequest(ctx, deviceID)
	return err
}

func (capsrv *ManageProvisioningCapnpServer) GetApprovedRequests(
	ctx context.Context, call hubapi.CapManageProvisioning_getApprovedRequests) error {

	statusList, err := capsrv.pogosrv.GetApprovedRequests(ctx)
	if err == nil {
		res, err2 := call.AllocResults()
		err = err2
		if err2 == nil {
			statusListCapnp := capserializer.MarshalProvStatusList(statusList)
			res.SetRequests(statusListCapnp)
		}
	}
	return err
}

func (capsrv *ManageProvisioningCapnpServer) GetPendingRequests(
	ctx context.Context, call hubapi.CapManageProvisioning_getPendingRequests) error {

	statusList, err := capsrv.pogosrv.GetPendingRequests(ctx)
	if err == nil {
		res, err2 := call.AllocResults()
		err = err2
		if err2 == nil {
			statusListCapnp := capserializer.MarshalProvStatusList(statusList)
			res.SetRequests(statusListCapnp)
		}
	}
	return err
}
