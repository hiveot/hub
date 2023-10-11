package capnpserver

import (
	"context"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/pkg/history"
)

// ReadHistoryCapnpServer is a capnproto server adapter for the history server
// This implements the hubapi.ReadHistory interface
type ReadHistoryCapnpServer struct {
	svc history.IReadHistory
}

// GetEventHistory returns a cursor to iterate the history of the thing
// name is the event or action to filter on. Use "" to iterate all events/action of the thing
// The cursor MUST be released after use.
func (capsrv *ReadHistoryCapnpServer) GetEventHistory(
	ctx context.Context, call hubapi.CapReadHistory_getEventHistory) error {

	args := call.Args()
	publisherID, _ := args.PublisherID()
	thingID, _ := args.ThingID()
	eventName, _ := args.Name()
	cursor := capsrv.svc.GetEventHistory(ctx, publisherID, thingID, eventName)

	cursorSrv := NewHistoryCursorCapnpServer(cursor)
	capnpCursorServer := hubapi.CapHistoryCursor_ServerToClient(cursorSrv)

	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCursor(capnpCursorServer)
	}
	return err
}

// GetProperties returns the most recent property and event values of the Thing
//
//	names is the list of properties to return. Use "" to return all known properties.
func (capsrv *ReadHistoryCapnpServer) GetProperties(
	ctx context.Context, call hubapi.CapReadHistory_getProperties) error {

	args := call.Args()
	publisherID, _ := args.PublisherID()
	thingID, _ := args.ThingID()
	capNameList, _ := args.Names()
	names := caphelp.UnmarshalStringList(capNameList)
	valueList := capsrv.svc.GetProperties(ctx, publisherID, thingID, names)

	res, err := call.AllocResults()
	if err == nil {
		valueListCapnp := caphelp.MarshalThingValueList(valueList)
		err = res.SetValueList(valueListCapnp)
	}
	return err
}

//func (capsrv *ReadHistoryCapnpServer) Info(
//	ctx context.Context, call hubapi.CapReadHistory_info) error {
//	bucketInfo := capsrv.svc.Info(ctx)
//	res, err := call.AllocResults()
//	if err == nil {
//		_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
//		infoCapnp, _ := hubapi.NewBucketStoreInfo(seg)
//		infoCapnp.SetDataSize(bucketInfo.DataSize)
//		_ = infoCapnp.SetEngine(bucketInfo.Engine)
//		_ = infoCapnp.SetId(bucketInfo.Id)
//		infoCapnp.SetNrRecords(bucketInfo.NrRecords)
//		err = res.SetInfo(infoCapnp)
//	}
//	return err
//}

func (capsrv *ReadHistoryCapnpServer) Shutdown() {
	// Release on the client calls capnp release
	// Pass this to the server to cleanup
	capsrv.svc.Release()
}
