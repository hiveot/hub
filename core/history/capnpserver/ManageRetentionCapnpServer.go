package capnpserver

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/history/capserializer"
)

// ManageRetentionCapnpServer is a capnproto adapter for the history store
// This implements the capnproto generated interface ManageRetention_Server
// See hub/api/go/hubapi/History.capnp.go for the interface.
type ManageRetentionCapnpServer struct {
	svc history.IManageRetention
}

func (capsrv *ManageRetentionCapnpServer) GetEvents(
	ctx context.Context, call hubapi.CapManageRetention_getEvents) error {

	evList, _ := capsrv.svc.GetEvents(ctx)
	logrus.Infof("%v", evList)
	capRetList := capserializer.MarshalRetList(evList)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetRetList(capRetList)
	}
	return err
}

func (capsrv *ManageRetentionCapnpServer) GetEventRetention(
	ctx context.Context, call hubapi.CapManageRetention_getEventRetention) error {

	args := call.Args()
	eventName, _ := args.Name()
	evRet, _ := capsrv.svc.GetEventRetention(ctx, eventName)

	capEvRet := capserializer.MarshalEventRetention(evRet)
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetRet(capEvRet)
	}
	return err
}

func (capsrv *ManageRetentionCapnpServer) RemoveEventRetention(
	ctx context.Context, call hubapi.CapManageRetention_removeEventRetention) error {

	args := call.Args()
	eventName, _ := args.Name()
	err := capsrv.svc.RemoveEventRetention(ctx, eventName)
	return err
}

func (capsrv *ManageRetentionCapnpServer) SetEventRetention(
	ctx context.Context, call hubapi.CapManageRetention_setEventRetention) error {

	args := call.Args()
	capEvRet, _ := args.Ret()
	evRet := capserializer.UnmarshalEventRetention(capEvRet)
	err := capsrv.svc.SetEventRetention(ctx, evRet)
	return err
}

func (capsrv *ManageRetentionCapnpServer) TestEvent(
	ctx context.Context, call hubapi.CapManageRetention_testEvent) error {

	args := call.Args()
	capThingValue, _ := args.Tv()
	tv := caphelp.UnmarshalThingValue(capThingValue)
	pass, err := capsrv.svc.TestEvent(ctx, tv)
	if err == nil {
		res, err2 := call.AllocResults()
		err = err2
		res.SetRetained(pass)
	}
	return err
}
