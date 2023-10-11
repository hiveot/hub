// Package capnpclient that wraps the capnp generated client with a POGS API
package capnpclient

import (
	"context"

	"github.com/hiveot/hub/lib/caphelp"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/thing"
)

// AddHistoryCapnpClient provides a POGS wrapper around the capnp client API
// This implements the IUpdateHistory interface
type AddHistoryCapnpClient struct {
	capability hubapi.CapAddHistory // capnp client
}

// AddAction adds a Thing action with the given name and value to the action history
// TODO: split this into get capability and add action
func (cl *AddHistoryCapnpClient) AddAction(ctx context.Context, actionValue thing.ThingValue) error {

	// next add the action
	method, release := cl.capability.AddAction(ctx,
		func(params hubapi.CapAddHistory_addAction_Params) error {
			capValue := caphelp.MarshalThingValue(actionValue)
			err2 := params.SetTv(capValue)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

// AddEvent adds an event to the event history
func (cl *AddHistoryCapnpClient) AddEvent(
	ctx context.Context, eventValue thing.ThingValue) error {

	method, release := cl.capability.AddEvent(ctx,
		func(params hubapi.CapAddHistory_addEvent_Params) error {
			capValue := caphelp.MarshalThingValue(eventValue)
			err2 := params.SetTv(capValue)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

func (cl *AddHistoryCapnpClient) AddEvents(
	ctx context.Context, events []thing.ThingValue) error {

	method, release := cl.capability.AddEvents(ctx,
		func(params hubapi.CapAddHistory_addEvents_Params) error {
			// suspect that this conversion is slow
			capValues := caphelp.MarshalThingValueList(events)
			err2 := params.SetTv(capValues)
			return err2
		})
	defer release()
	_, err := method.Struct()
	return err
}

func (cl *AddHistoryCapnpClient) Release() {
	cl.capability.Release()
}

// NewAddHistoryCapnpClient returns an update-history client using the capnp protocol
// Intended for internal use.
func NewAddHistoryCapnpClient(cap hubapi.CapAddHistory) *AddHistoryCapnpClient {
	cl := &AddHistoryCapnpClient{capability: cap}
	return cl
}
