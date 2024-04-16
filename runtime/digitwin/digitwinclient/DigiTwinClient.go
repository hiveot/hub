package digitwinclient

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// DigiTwinClient is the golang client for talking to the digitwin server.
// This marshals method parameters and unmarshals the result.
type DigiTwinClient struct {
	pm api.IPostActionMessage
}

func (cl *DigiTwinClient) ReadActions(thingID string, keys []string) (actions things.ThingMessageMap, err error) {
	args := api.ReadActionsArgs{ThingID: thingID, Keys: keys}
	resp := api.ReadActionsResp{}

	err = cl.pm(api.DigiTwinServiceID, api.ReadActionsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Actions, nil
}

func (cl *DigiTwinClient) ReadEvents(thingID string, keys []string) (events things.ThingMessageMap, err error) {
	args := api.ReadEventsArgs{ThingID: thingID, Keys: keys}
	resp := api.ReadEventsResp{}

	err = cl.pm(api.DigiTwinServiceID, api.ReadEventsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Events, nil
}

func (cl *DigiTwinClient) ReadProperties(thingID string, keys []string) (props things.ThingMessageMap, err error) {
	args := api.ReadPropertiesArgs{ThingID: thingID, Keys: keys}
	resp := api.ReadPropertiesResp{}

	err = cl.pm(api.DigiTwinServiceID, api.ReadPropertiesMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Props, nil
}

func (cl *DigiTwinClient) ReadThing(thingID string) (td *things.TD, err error) {
	args := api.ReadThingArgs{ThingID: thingID}
	resp := api.ReadThingResp{}

	err = cl.pm(api.DigiTwinServiceID, api.ReadThingMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.TD, nil
}

func (cl *DigiTwinClient) ReadThings(offset, limit int) (tdList []*things.TD, err error) {
	args := api.ReadThingsArgs{Offset: offset, Limit: limit}
	resp := api.ReadThingsResp{}

	err = cl.pm(api.DigiTwinServiceID, api.ReadThingsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.TDs, nil
}

func (cl *DigiTwinClient) RemoveThing(thingID string) (err error) {
	args := api.RemoveThingArgs{ThingID: thingID}

	err = cl.pm(api.DigiTwinServiceID, api.RemoveThingMethod, &args, nil)
	return err
}

// NewDigiTwinClient creates a new instance of the digitwin client using
// the given hub connection.
func NewDigiTwinClient(pm IPostActionMessage) *DigiTwinClient {
	cl := DigiTwinClient{pm: pm}
	return &cl
}
