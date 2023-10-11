package capnpserver

import (
	"context"

	"github.com/hiveot/hub/lib/caphelp"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/history"
)

// AddHistoryCapnpServer provides the capnp RPC server for adding to the history
type AddHistoryCapnpServer struct {
	svc history.IAddHistory
	// TODO: restrict to a specific device publisher
}

func (capsrv *AddHistoryCapnpServer) AddAction(
	ctx context.Context, call hubapi.CapAddHistory_addAction) error {

	args := call.Args()
	capValue, _ := args.Tv()
	publisherID, _ := capValue.PublisherID()
	thingID, _ := capValue.ThingID()
	name, _ := capValue.Name()
	valueJSON, _ := capValue.Data()
	created, _ := capValue.Created()
	actionValue := thing.NewThingValue(publisherID, thingID, name, valueJSON)
	actionValue.Created = created
	err := capsrv.svc.AddAction(ctx, actionValue)
	return err
}

func (capsrv *AddHistoryCapnpServer) AddEvent(
	ctx context.Context, call hubapi.CapAddHistory_addEvent) error {

	args := call.Args()
	capValue, _ := args.Tv()
	publisherID, _ := capValue.PublisherID()
	thingID, _ := capValue.ThingID()
	name, _ := capValue.Name()
	valueJSON, _ := capValue.Data()
	created, _ := capValue.Created()
	eventValue := thing.NewThingValue(publisherID, thingID, name, valueJSON)
	eventValue.Created = created

	err := capsrv.svc.AddEvent(ctx, eventValue)
	//call.Ack()
	return err
}
func (capsrv *AddHistoryCapnpServer) AddEvents(
	ctx context.Context, call hubapi.CapAddHistory_addEvents) error {

	args := call.Args()
	capValues, _ := args.Tv()
	eventValues := caphelp.UnmarshalThingValueList(capValues)
	err := capsrv.svc.AddEvents(ctx, eventValues)
	return err
}

func (capsrv *AddHistoryCapnpServer) Shutdown() {
	// Release on the client calls capnp release
	// Pass this to the server to cleanup
	capsrv.svc.Release()
}
