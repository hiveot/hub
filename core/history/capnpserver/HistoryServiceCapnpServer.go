package capnpserver

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/resolver/capprovider"
)

// HistoryServiceCapnpServer is a capnproto adapter for the history store
// This implements the capnproto generated interface History_Server
// See hub/api/go/hubapi/HistoryStore.capnp.go for the interface.
type HistoryServiceCapnpServer struct {
	svc history.IHistoryService
}

func (capsrv *HistoryServiceCapnpServer) CapAddHistory(
	ctx context.Context, call hubapi.CapHistoryService_capAddHistory) error {
	// create a client instance for adding history
	args := call.Args()
	clientID, _ := args.ClientID()
	ignoreRetention := args.IgnoreRetention()
	capAny, _ := capsrv.svc.CapAddHistory(ctx, clientID, ignoreRetention)
	ahCapSrv := &AddHistoryCapnpServer{
		svc: capAny,
	}
	// reuse the add history marshalling
	capnpAddHistory := hubapi.CapAddHistory_ServerToClient(ahCapSrv)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCap(capnpAddHistory)
	}
	return err
}

func (capsrv *HistoryServiceCapnpServer) CapManageRetention(
	ctx context.Context, call hubapi.CapHistoryService_capManageRetention) error {
	args := call.Args()
	clientID, _ := args.ClientID()
	capManageRet, _ := capsrv.svc.CapManageRetention(ctx, clientID)
	manageRetCapnpServer := &ManageRetentionCapnpServer{
		svc: capManageRet,
	}
	// reuse the add history marshalling
	capnpManageRet := hubapi.CapManageRetention_ServerToClient(manageRetCapnpServer)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCap(capnpManageRet)
	}
	return err
}
func (capsrv *HistoryServiceCapnpServer) CapReadHistory(
	ctx context.Context, call hubapi.CapHistoryService_capReadHistory) error {

	// create a client instance for reading the history
	args := call.Args()
	clientID, _ := args.ClientID()
	capRead, _ := capsrv.svc.CapReadHistory(ctx, clientID)
	readSrv := &ReadHistoryCapnpServer{
		svc: capRead,
	}
	capnpReadHistory := hubapi.CapReadHistory_ServerToClient(readSrv)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCap(capnpReadHistory)
	}
	return err
}

//func (capsrv *HistoryServiceCapnpServer) Info(
//	ctx context.Context, call hubapi.CapHistoryService_info) (err error) {
//
//	inf, err := capsrv.svc.Info(ctx)
//	if err == nil {
//		res, err2 := call.AllocResults()
//		err = err2
//		storeInfo, _ := res.NewStatistics()
//		storeInfo.SetNrActions(int64(inf.NrActions))
//		storeInfo.SetNrEvents(int64(inf.NrEvents))
//		storeInfo.SetEngine(inf.Engine)
//		storeInfo.SetUptime(int64(inf.Uptime))
//	}
//
//	return err
//}

// StartHistoryServiceCapnpServer returns the capnp protocol server for the history store
func StartHistoryServiceCapnpServer(svc history.IHistoryService, lis net.Listener) (err error) {
	serviceName := history.ServiceName

	capsrv := &HistoryServiceCapnpServer{
		svc: svc,
	}
	// the provider serves the exported capabilities
	// this replaces CapHistoryService_ServerToClient
	capProv := capprovider.NewCapServer(
		serviceName, hubapi.CapHistoryService_Methods(nil, capsrv))

	capProv.ExportCapability(hubapi.CapNameAddHistory, []string{hubapi.AuthTypeService})

	capProv.ExportCapability(hubapi.CapNameManageRetention, []string{hubapi.AuthTypeService})

	capProv.ExportCapability(hubapi.CapNameReadHistory,
		[]string{hubapi.AuthTypeService, hubapi.AuthTypeUser})

	logrus.Infof("Starting '%s' service capnp adapter listening on: %s", serviceName, lis.Addr())
	err = capProv.Start(lis)

	return err
}
