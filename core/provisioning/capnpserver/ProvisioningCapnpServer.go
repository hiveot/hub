package capnpserver

import (
	"context"
	"github.com/hiveot/hub/pkg/provisioning"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/pkg/resolver/capprovider"
)

// ProvisioningCapnpServer provides the capnproto RPC server for IOT device provisioning.
// This implements the capnproto generated interface Provisioning_Server
// See hub/api/go/hubapi/Provisioning.capnp.go for the interface.
type ProvisioningCapnpServer struct {
	// the plain-old-go-object provisioning server
	svc provisioning.IProvisioning
}

func (capsrv *ProvisioningCapnpServer) CapManageProvisioning(
	ctx context.Context, call hubapi.CapProvisioning_capManageProvisioning) error {

	clientID, _ := call.Args().ClientID()
	// create the service instance for this request
	capManage, _ := capsrv.svc.CapManageProvisioning(ctx, clientID)
	mngCapSrv := &ManageProvisioningCapnpServer{
		pogosrv: capManage,
	}

	// wrap it with a capnp proxy
	capability := hubapi.CapManageProvisioning_ServerToClient(mngCapSrv)
	res, err := call.AllocResults()
	if err == nil {
		// return the proxy
		err = res.SetCap(capability)
	}
	return err
}

func (capsrv *ProvisioningCapnpServer) CapRefreshProvisioning(
	ctx context.Context, call hubapi.CapProvisioning_capRefreshProvisioning) error {

	clientID, _ := call.Args().ClientID()
	// create the service instance for this request
	capRefresh, _ := capsrv.svc.CapRefreshProvisioning(ctx, clientID)

	// TODO: restrict it to the deviceID of the caller
	refreshCapSrv := &RefreshProvisioningCapnpServer{
		pogosrv: capRefresh,
	}

	// wrap it with a capnp proxy
	capability := hubapi.CapRefreshProvisioning_ServerToClient(refreshCapSrv)
	res, err := call.AllocResults()
	if err == nil {
		// return the proxy
		err = res.SetCap(capability)
	}
	return err
}
func (capsrv *ProvisioningCapnpServer) CapRequestProvisioning(
	ctx context.Context, call hubapi.CapProvisioning_capRequestProvisioning) error {

	clientID, _ := call.Args().ClientID()
	// create the service instance for this request
	capRequest, _ := capsrv.svc.CapRequestProvisioning(ctx, clientID)
	reqCapSrv := &RequestProvisioningCapnpServer{
		pogosrv: capRequest,
	}

	// wrap it with a capnp proxy
	capability := hubapi.CapRequestProvisioning_ServerToClient(reqCapSrv)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCap(capability)
	}
	return err
}

// StartProvisioningCapnpServer starts the capnp server for the provisioning service
func StartProvisioningCapnpServer(svc provisioning.IProvisioning, lis net.Listener) error {
	serviceName := provisioning.ServiceName

	srv := &ProvisioningCapnpServer{
		svc: svc,
	}
	capProv := capprovider.NewCapServer(
		serviceName, hubapi.CapProvisioning_Methods(nil, srv))

	capProv.ExportCapability("capManageProvisioning",
		[]string{hubapi.AuthTypeService})

	capProv.ExportCapability("capRequestProvisioning",
		[]string{hubapi.AuthTypeService, hubapi.AuthTypeIotDevice})

	capProv.ExportCapability("capRefreshProvisioning",
		[]string{hubapi.AuthTypeService, hubapi.AuthTypeIotDevice})

	logrus.Infof("Starting provisioning service capnp adapter on: %s", lis.Addr())
	err := capProv.Start(lis)
	return err
}
