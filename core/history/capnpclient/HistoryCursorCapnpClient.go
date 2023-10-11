// Package capnpclient that wraps the capnp generated client with a POGS API
package capnpclient

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/lib/thing"
)

// HistoryCursorCapnpClient provides a POGS wrapper around the capnp client API
// This implements the IHistoryCursor interface
type HistoryCursorCapnpClient struct {
	capability hubapi.CapHistoryCursor // capnp client
}

// First positions the cursor at the first key in the ordered list
func (cl *HistoryCursorCapnpClient) First() (thingValue thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.First(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		tvCapnp, _ := resp.Tv()
		valid = resp.Valid()
		thingValue = caphelp.UnmarshalThingValue(tvCapnp)
	}
	return thingValue, valid
}

// Last positions the cursor at the last key in the ordered list
func (cl *HistoryCursorCapnpClient) Last() (thingValue thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.Last(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		tvCapnp, _ := resp.Tv()
		valid = resp.Valid()
		thingValue = caphelp.UnmarshalThingValue(tvCapnp)
	}
	return thingValue, valid
}

// Next moves the cursor to the next key from the current cursor
func (cl *HistoryCursorCapnpClient) Next() (thingValue thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.Next(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		tvCapnp, _ := resp.Tv()
		valid = resp.Valid()
		thingValue = caphelp.UnmarshalThingValue(tvCapnp)
	}
	return thingValue, valid
}

// NextN moves the cursor to the next N steps from the current cursor
func (cl *HistoryCursorCapnpClient) NextN(steps uint) (batch []thing.ThingValue, valid bool) {
	ctx := context.Background()

	method, release := cl.capability.NextN(ctx,
		func(params hubapi.CapHistoryCursor_nextN_Params) error {
			params.SetSteps(uint32(steps))
			return nil
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		valid = resp.Valid()
		thingValueListCap, _ := resp.Batch()
		batch = caphelp.UnmarshalThingValueList(thingValueListCap)
	}
	return batch, valid
}

// Prev moves the cursor to the previous key from the current cursor
func (cl *HistoryCursorCapnpClient) Prev() (thingValue thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.Prev(ctx, nil)
	defer release()
	resp, err := method.Struct()
	if err == nil {
		tvCapnp, _ := resp.Tv()
		thingValue = caphelp.UnmarshalThingValue(tvCapnp)
		valid = resp.Valid()
	}
	return thingValue, valid
}

// PrevN moves the cursor back N steps from the current cursor
func (cl *HistoryCursorCapnpClient) PrevN(steps uint) (batch []thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.PrevN(ctx,
		func(params hubapi.CapHistoryCursor_prevN_Params) error {
			params.SetSteps(uint32(steps))
			return nil
		})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		valid = resp.Valid()
		thingValueListCap, _ := resp.Batch()
		batch = caphelp.UnmarshalThingValueList(thingValueListCap)
	}
	return batch, valid
}

// Release the cursor capability
func (cl *HistoryCursorCapnpClient) Release() {
	logrus.Infof("releasing bucket cursor")
	cl.capability.Release()
}

// Seek the starting point for iterating the history
func (cl *HistoryCursorCapnpClient) Seek(isoTimestamp string) (thingValue thing.ThingValue, valid bool) {
	ctx := context.Background()
	method, release := cl.capability.Seek(ctx, func(params hubapi.CapHistoryCursor_seek_Params) error {
		err2 := params.SetIsoTimestamp(isoTimestamp)
		return err2
	})
	defer release()
	resp, err := method.Struct()
	if err == nil {
		valid = resp.Valid()
		tvCapnp, _ := resp.Tv()
		thingValue = caphelp.UnmarshalThingValue(tvCapnp)
	}
	return thingValue, valid
}

//func (cl *HistoryCursorCapnpClient) Info(
//	ctx context.Context) (info history.StoreInfo, err error) {
//
//	method, release := cl.capability.Info(ctx, nil)
//	defer release()
//	resp, err := method.Struct()
//	if err == nil {
//		capInfo, _ := resp.Statistics()
//		engine, _ := capInfo.Engine()
//		nrActions := capInfo.NrActions()
//		nrEvents := capInfo.NrEvents()
//		uptimeSec := capInfo.Uptime()
//		info = history.StoreInfo{
//			Engine:    engine,
//			NrActions: int(nrActions),
//			NrEvents:  int(nrEvents),
//			Uptime:    int(uptimeSec),
//		}
//	}
//	return info, err
//}

// NewHistoryCursorCapnpClient returns a read history client using the capnp protocol
// Intended for internal use.
func NewHistoryCursorCapnpClient(cap hubapi.CapHistoryCursor) *HistoryCursorCapnpClient {
	cl := &HistoryCursorCapnpClient{capability: cap}
	return cl
}
