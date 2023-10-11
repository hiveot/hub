package capnpclient

import (
	"context"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/history"
)

// ReadHistoryCapnpClient capnp client for making RPC calls to read a thing's history
type ReadHistoryCapnpClient struct {
	capability hubapi.CapReadHistory
}

func (cl *ReadHistoryCapnpClient) GetEventHistory(
	ctx context.Context, publisherID, thingID string, name string) history.IHistoryCursor {

	method, release := cl.capability.GetEventHistory(ctx,
		func(params hubapi.CapReadHistory_getEventHistory_Params) error {
			err := params.SetName(name)
			_ = params.SetPublisherID(publisherID)
			_ = params.SetThingID(thingID)
			return err
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		cursor := resp.Cursor().AddRef()
		return NewHistoryCursorCapnpClient(cursor)
	}
	return nil
}

func (cl *ReadHistoryCapnpClient) GetProperties(
	ctx context.Context, publisherID, thingID string, names []string) (values []thing.ThingValue) {

	method, release := cl.capability.GetProperties(ctx,
		func(params hubapi.CapReadHistory_getProperties_Params) error {
			_ = params.SetPublisherID(publisherID)
			_ = params.SetThingID(thingID)
			nameList := caphelp.MarshalStringList(names)
			err := params.SetNames(nameList)
			return err
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		tlist, _ := resp.ValueList()
		values = caphelp.UnmarshalThingValueList(tlist)
		return values
	}
	return nil
}

// Info returns the history storage information of the thing
//func (cl *ReadHistoryCapnpClient) Info(
//	ctx context.Context) (bucketInfo *bucketstore.BucketStoreInfo) {
//
//	bucketInfo = &bucketstore.BucketStoreInfo{}
//	method, release := cl.capability.Info(ctx, nil)
//	defer release()
//	resp, err := method.Struct()
//	if err == nil {
//		infoCapnp, _ := resp.Info()
//		bucketEngine, _ := infoCapnp.Engine()
//		bucketID, _ := infoCapnp.Id()
//		bucketInfo.Id = bucketID
//		bucketInfo.DataSize = infoCapnp.DataSize()
//		bucketInfo.Engine = bucketEngine
//		bucketInfo.NrRecords = infoCapnp.NrRecords()
//	}
//	return
//}

func (cl *ReadHistoryCapnpClient) Release() {
	cl.capability.Release()
}

func NewReadHistoryCapnpClient(capability hubapi.CapReadHistory) *ReadHistoryCapnpClient {
	cl := &ReadHistoryCapnpClient{capability: capability}
	return cl
}
