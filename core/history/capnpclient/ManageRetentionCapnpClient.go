package capnpclient

import (
	"context"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/history/capserializer"
)

// ManageRetentionCapnpClient capnp client for managing retention rules
type ManageRetentionCapnpClient struct {
	capability hubapi.CapManageRetention
}

func (cl *ManageRetentionCapnpClient) GetEvents(
	ctx context.Context) ([]history.EventRetention, error) {

	method, release := cl.capability.GetEvents(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		capRetList, err := resp.RetList()
		retList := capserializer.UnmarshalRetList(capRetList)
		return retList, err
	}
	return nil, err
}

func (cl *ManageRetentionCapnpClient) GetEventRetention(
	ctx context.Context, eventName string) (ret history.EventRetention, err error) {

	method, release := cl.capability.GetEventRetention(ctx,
		func(params hubapi.CapManageRetention_getEventRetention_Params) error {
			err2 := params.SetName(eventName)
			return err2
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		capRet, err2 := resp.Ret()
		err = err2
		ret = capserializer.UnmarshalEventRetention(capRet)
	}
	return ret, err
}

func (cl *ManageRetentionCapnpClient) RemoveEventRetention(ctx context.Context, name string) error {
	method, release := cl.capability.RemoveEventRetention(ctx,
		func(params hubapi.CapManageRetention_removeEventRetention_Params) error {
			err2 := params.SetName(name)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

func (cl *ManageRetentionCapnpClient) SetEventRetention(
	ctx context.Context, ret history.EventRetention) error {

	method, release := cl.capability.SetEventRetention(ctx,
		func(params hubapi.CapManageRetention_setEventRetention_Params) error {
			capRet := capserializer.MarshalEventRetention(ret)
			err2 := params.SetRet(capRet)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

func (cl *ManageRetentionCapnpClient) TestEvent(
	ctx context.Context, tv thing.ThingValue) (retained bool, err error) {

	method, release := cl.capability.TestEvent(ctx,
		func(params hubapi.CapManageRetention_testEvent_Params) error {
			capTv := caphelp.MarshalThingValue(tv)
			err2 := params.SetTv(capTv)
			return err2
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		retained = resp.Retained()
	}
	return retained, err
}

func (cl *ManageRetentionCapnpClient) Release() {
	cl.capability.Release()
}

func NewManageRetentionCapnpClient(capability hubapi.CapManageRetention) *ManageRetentionCapnpClient {
	cl := &ManageRetentionCapnpClient{capability: capability}
	return cl
}
