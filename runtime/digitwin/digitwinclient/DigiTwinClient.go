package digitwinclient

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

func ReadActions(mt api.IMessageTransport,
	thingID string, keys []string, since string) (
	actions things.ThingMessageMap, err error) {

	args := api.ReadActionsArgs{ThingID: thingID, Keys: keys, Since: since}
	resp := api.ReadActionsResp{}

	err = mt(api.DigiTwinThingID, api.ReadActionsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func ReadActionHistory(mt api.IMessageTransport,
	thingID string, actionKey string, start string, end string) (
	actions []*things.ThingMessage, err error) {

	args := api.ReadActionHistoryArgs{
		ThingID: thingID,
		Key:     actionKey,
		Start:   start,
		End:     end,
	}
	resp := api.ReadActionHistoryResp{}

	err = mt(api.DigiTwinThingID, api.ReadActionHistoryMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}
func ReadEvents(mt api.IMessageTransport, thingID string, keys []string, since string) (
	events things.ThingMessageMap, err error) {

	args := api.ReadEventsArgs{ThingID: thingID, Keys: keys, Since: since}
	resp := api.ReadEventsResp{}

	err = mt(api.DigiTwinThingID, api.ReadEventsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func ReadEventHistory(mt api.IMessageTransport,
	thingID string, eventKey string, start string, end string) (
	events []*things.ThingMessage, err error) {

	args := api.ReadEventHistoryArgs{
		ThingID: thingID,
		Key:     eventKey,
		Start:   start,
		End:     end,
	}
	resp := api.ReadEventHistoryResp{}

	err = mt(api.DigiTwinThingID, api.ReadEventHistoryMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func ReadProperties(mt api.IMessageTransport,
	thingID string, keys []string, since string) (props things.ThingMessageMap, err error) {

	args := api.ReadPropertiesArgs{ThingID: thingID, Keys: keys, Since: since}
	resp := api.ReadPropertiesResp{}

	err = mt(api.DigiTwinThingID, api.ReadPropertiesMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func ReadThing(mt api.IMessageTransport, thingID string) (td *things.TD, err error) {
	args := api.ReadThingArgs{ThingID: thingID}
	resp := api.ReadThingResp{}

	err = mt(api.DigiTwinThingID, api.ReadThingMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.TD, nil
}

func ReadThings(mt api.IMessageTransport, offset, limit int) (tdList []*things.TD, err error) {
	args := api.ReadThingsArgs{Offset: offset, Limit: limit}
	resp := api.ReadThingsResp{}

	err = mt(api.DigiTwinThingID, api.ReadThingsMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	return resp.TDs, nil
}

func RemoveThing(mt api.IMessageTransport, thingID string) (err error) {
	args := api.RemoveThingArgs{ThingID: thingID}

	err = mt(api.DigiTwinThingID, api.RemoveThingMethod, &args, nil)
	return err
}

// WriteAction requests to trigger a thing action
//
//	actionID is the ID of the action request
//	status is the current delivery status of the request
//	value is the response value if status is delivered and a response is provided
//	err is the error received
func WriteAction(mt api.IMessageTransport,
	thingID string, key string, rawData []byte) (actionID string, status string, value []byte, err error) {

	args := api.WriteActionArgs{ThingID: thingID, Key: key, Value: rawData}
	resp := api.WriteActionResp{}

	err = mt(api.DigiTwinThingID, api.WriteActionMethod, &args, &resp)
	return resp.ActionID, resp.Status, resp.Value, err
}

// WriteProperty requests to update a property value
// the value is a stringified value based on the TD
func WriteProperty(mt api.IMessageTransport, thingID string, key string, value string) (err error) {

	args := api.WritePropertyArgs{ThingID: thingID, Key: key, Value: value}

	err = mt(api.DigiTwinThingID, api.WritePropertyMethod, &args, nil)
	return err
}

// WriteThing updates a thing TD document
func WriteThing(mt api.IMessageTransport, td *things.TD) (err error) {

	args := api.WriteThingArgs{TD: td}

	err = mt(api.DigiTwinThingID, api.WriteActionMethod, &args, nil)
	return err
}
