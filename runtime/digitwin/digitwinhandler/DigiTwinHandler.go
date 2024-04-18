package digitwinhandler

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/router"
	"log/slog"
)

// DigiTwinHandler serves the message based interface to the digitwin service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type DigiTwinHandler struct {
	svc *service.DigiTwinService
}

// HandleMessage an event or action message for the digital twin service
func (h *DigiTwinHandler) HandleMessage(msg *things.ThingMessage) (reply []byte, err error) {
	if msg.MessageType == vocab.MessageTypeEvent {
		if msg.Key == vocab.EventTypeTD {
			err = h.HandleTDEvent(msg)
		} else if msg.Key == vocab.EventTypeProperties {
			err = h.svc.Values.HandleEvent(msg)
		} else {
			err = h.svc.Values.HandleEvent(msg)
		}
		return nil, err
	} else if msg.MessageType == vocab.MessageTypeAction {
		// Methods for accessing Thing description documents
		switch msg.Key {
		case api.ReadThingMethod:
			return h.HandleReadThing(msg)
		case api.ReadThingsMethod:
			return h.HandleReadThings(msg)
		case api.RemoveThingMethod:
			return h.HandleRemoveThing(msg)

		// Methods for accessing Thing values
		case api.ReadActionsMethod:
			return h.HandleReadActions(msg)
		case api.ReadEventsMethod:
			return h.HandleReadEvents(msg)
		case api.ReadPropertiesMethod:
			return h.HandleReadProperties(msg)
		}
	}
	return nil, nil
}

// HandleReadThings handles an action request for a list of TD documents
func (h *DigiTwinHandler) HandleReadThings(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadThingsArgs
	var tds []*things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tds, err = h.svc.Directory.ReadThings(args.Offset, args.Limit)
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
func (h *DigiTwinHandler) HandleReadThing(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadThingArgs
	var tdd *things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tdd, err = h.svc.Directory.ReadThing(args.ThingID)
	}
	if err != nil {
		return nil, err
	}
	resp := api.ReadThingResp{TD: tdd}
	respJson, err := json.Marshal(resp)
	return respJson, err
}

// HandleRemoveThing handles an action request for removing a TD document
func (h *DigiTwinHandler) HandleRemoveThing(msg *things.ThingMessage) ([]byte, error) {
	var args api.RemoveThingArgs

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.Directory.RemoveThing(msg.SenderID, args.ThingID)
	}
	if err != nil {
		return nil, err
	}
	return nil, err
}

func (h *DigiTwinHandler) HandleReadActions(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadActionsArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadActions. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadActions(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadActionsResp{Actions: values}
	reply, err := json.Marshal(resp)
	return reply, err
}
func (h *DigiTwinHandler) HandleReadEvents(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadEventsArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadEvents. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadEvents(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadEventsResp{Events: values}
	reply, err := json.Marshal(resp)
	return reply, err
}
func (h *DigiTwinHandler) HandleReadProperties(msg *things.ThingMessage) ([]byte, error) {
	var args api.ReadPropertiesArgs
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadProperties. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadProperties(args.ThingID, args.Keys)
	if err != nil {
		return nil, err
	}
	resp := api.ReadPropertiesResp{Props: values}
	reply, err := json.Marshal(resp)
	return reply, err
}

// HandleTDEvent handles an event containing a TD document
func (h *DigiTwinHandler) HandleTDEvent(msg *things.ThingMessage) error {
	var args *things.TD
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.Directory.UpdateThing(msg.SenderID, msg.ThingID, args)
	}
	if err != nil {
		slog.Warn("HandleEvent: TD update failed", "err", err)
	}
	return err
}

// NewDigiTwinHandler creates a new instance of the messaging api server for the
// digitwin service.
func NewDigiTwinHandler(svc *service.DigiTwinService) router.MessageHandler {
	decoder := DigiTwinHandler{svc: svc}
	return decoder.HandleMessage
}
