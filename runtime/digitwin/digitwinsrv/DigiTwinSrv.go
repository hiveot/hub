package digitwinsrv

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"log/slog"
)

// DigiTwinSrv serves the message based interface to the digitwin service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type DigiTwinSrv struct {
	svc *service.DigiTwinService
}

// HandleMessage an event or action message for the digital twin service
func (rpc *DigiTwinSrv) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeEvent && msg.Key == vocab.EventTypeTD {
		return nil, rpc.HandleTDEvent(msg)
	} else if msg.MessageType == vocab.MessageTypeAction {
		// Methods for accessing Thing description documents
		switch msg.Key {
		case api.ReadThingMethod:
			return rpc.HandleReadThing(msg)
		case api.ReadThingsMethod:
			return rpc.HandleReadThings(msg)
		case api.RemoveThingMethod:
			return rpc.HandleRemoveThing(msg)

		// Methods for accessing Thing values
		case api.ReadActionsMethod:
			return rpc.HandleReadActions(msg)
		case api.ReadEventsMethod:
			return rpc.HandleReadEvents(msg)
		case api.ReadPropertiesMethod:
			return rpc.HandleReadProperties(msg)
		}
	}
	return nil, nil
}

// HandleReadThings handles an action request for a list of TD documents
func (rpc *DigiTwinSrv) HandleReadThings(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadThingsArgs
	var tds []*things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tds, err = rpc.svc.Directory.ReadThings(args.Offset, args.Limit)
	}
	if err != nil {
		return nil, err
	}
	resp := api.ReadThingsResp{
		TDs: tds,
	}
	respJson, err := json.Marshal(resp)
	return respJson, err
}

// HandleReadThing handles an action request for a sing TD document
func (rpc *DigiTwinSrv) HandleReadThing(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadThingArgs
	var tdd *things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tdd, err = rpc.svc.Directory.ReadThing(args.ThingID)
	}
	if err != nil {
		return nil, err
	}
	resp := api.ReadThingResp{TD: tdd}
	respJson, err := json.Marshal(resp)
	return respJson, err
}

// HandleRemoveThing handles an action request for removing a TD document
func (rpc *DigiTwinSrv) HandleRemoveThing(msg *things.ThingMessage) ([]byte, error) {
	var args api.RemoveThingArgs

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = rpc.svc.Directory.RemoveThing(msg.SenderID, args.ThingID)
	}
	if err != nil {
		return nil, err
	}
	return nil, err
}

func (rpc *DigiTwinSrv) HandleReadActions(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadActionsArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadActions. Argument error: %w", err)
	}
	values, err := rpc.svc.Values.ReadActions(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadActionsResp{Actions: values}
	reply, err := json.Marshal(resp)
	return reply, err
}
func (rpc *DigiTwinSrv) HandleReadEvents(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadEventsArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadEvents. Argument error: %w", err)
	}
	values, err := rpc.svc.Values.ReadEvents(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadEventsResp{Events: values}
	reply, err := json.Marshal(resp)
	return reply, err
}
func (rpc *DigiTwinSrv) HandleReadProperties(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadPropertiesArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadProperties. Argument error: %w", err)
	}
	values, err := rpc.svc.Values.ReadProperties(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadPropertiesResp{Props: values}
	reply, err := json.Marshal(resp)
	return reply, err
}

// HandleTDEvent handles an event containing a TD document
func (rpc *DigiTwinSrv) HandleTDEvent(msg *things.ThingMessage) error {
	var args *things.TD
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = rpc.svc.Directory.UpdateThing(msg.SenderID, msg.ThingID, args)
	}
	if err != nil {
		slog.Warn("HandleEvent: TD update failed", "err", err)
	}
	return err
}

// NewDigiTwinSrv creates a new instance of the messaging api server for the
// digitwin service.
func NewDigiTwinSrv(svc *service.DigiTwinService) *DigiTwinSrv {
	rpc := DigiTwinSrv{svc: svc}
	return &rpc
}
