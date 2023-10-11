package capnpserver

import (
	"context"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/pkg/history"
)

// HistoryCursorCapnpServer is a capnproto RPC server for reading of the history store
type HistoryCursorCapnpServer struct {
	svc history.IHistoryCursor
}

func (capsrv HistoryCursorCapnpServer) First(
	_ context.Context, call hubapi.CapHistoryCursor_first) error {
	thingValue, valid := capsrv.svc.First()
	res, err := call.AllocResults()
	if err == nil {
		thingValueCapnp := caphelp.MarshalThingValue(thingValue)
		res.SetValid(valid)
		err = res.SetTv(thingValueCapnp)
	}
	return err
}

func (capsrv HistoryCursorCapnpServer) Last(
	_ context.Context, call hubapi.CapHistoryCursor_last) error {
	thingValue, valid := capsrv.svc.Last()
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueCapnp := caphelp.MarshalThingValue(thingValue)
		err = res.SetTv(thingValueCapnp)
	}
	return err
}

func (capsrv HistoryCursorCapnpServer) Next(
	_ context.Context, call hubapi.CapHistoryCursor_next) error {
	thingValue, valid := capsrv.svc.Next()
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueCapnp := caphelp.MarshalThingValue(thingValue)
		err = res.SetTv(thingValueCapnp)
	}
	return err
}

func (capsrv HistoryCursorCapnpServer) NextN(
	_ context.Context, call hubapi.CapHistoryCursor_nextN) error {
	args := call.Args()
	steps := args.Steps()
	thingValueList, valid := capsrv.svc.NextN(uint(steps))
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueListCapnp := caphelp.MarshalThingValueList(thingValueList)
		err = res.SetBatch(thingValueListCapnp)
	}
	return err
}

func (capsrv HistoryCursorCapnpServer) Prev(
	_ context.Context, call hubapi.CapHistoryCursor_prev) error {
	thingValue, valid := capsrv.svc.Prev()
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueCapnp := caphelp.MarshalThingValue(thingValue)
		err = res.SetTv(thingValueCapnp)
	}
	return err
}
func (capsrv HistoryCursorCapnpServer) PrevN(
	_ context.Context, call hubapi.CapHistoryCursor_prevN) error {
	args := call.Args()
	steps := args.Steps()
	thingValueList, valid := capsrv.svc.PrevN(uint(steps))
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueListCapnp := caphelp.MarshalThingValueList(thingValueList)
		err = res.SetBatch(thingValueListCapnp)
	}
	return err
}

func (capsrv HistoryCursorCapnpServer) Seek(
	_ context.Context, call hubapi.CapHistoryCursor_seek) error {
	args := call.Args()
	isoTimeStamp, _ := args.IsoTimestamp()
	thingValue, valid := capsrv.svc.Seek(isoTimeStamp)
	res, err := call.AllocResults()
	if err == nil {
		res.SetValid(valid)
		thingValueCapnp := caphelp.MarshalThingValue(thingValue)
		err = res.SetTv(thingValueCapnp)
	}
	return err
}

func (capsrv *HistoryCursorCapnpServer) Shutdown() {
	// Release on the client calls capnp Shutdown.
	// Pass this to the server to cleanup
	capsrv.svc.Release()
}

func NewHistoryCursorCapnpServer(cursor history.IHistoryCursor) *HistoryCursorCapnpServer {
	cursorCapnpServer := &HistoryCursorCapnpServer{
		svc: cursor,
	}
	return cursorCapnpServer
}
