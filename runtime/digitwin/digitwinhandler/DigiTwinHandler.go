package digitwinhandler

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
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
			// TDs update the directory
			err = h.HandleTDEvent(msg)
		} else {
			// Store property and other events
			err = h.svc.Values.HandleMessage(msg)
		}
		return nil, err
	} else if msg.MessageType == vocab.MessageTypeAction {
		// store action requests to things and services?
		_ = h.svc.Values.HandleMessage(msg)

		switch msg.Key {
		// Methods for writing an action to a Thing
		case api.WriteActionMethod:
			return h.HandleWriteAction(msg)

		// Methods for accessing Thing description documents
		case api.ReadThingMethod:
			return h.HandleReadThing(msg)
		case api.ReadThingsMethod:
			return h.HandleReadThings(msg)
		case api.RemoveThingMethod:
			return h.HandleRemoveThing(msg)

		// Method for reading the latest Thing values
		// Methods for accessing Thing values
		case api.ReadActionsMethod:
			return h.HandleReadActions(msg)
		case api.ReadEventsMethod:
			return h.HandleReadEvents(msg)
		case api.ReadPropertiesMethod:
			return h.HandleReadProperties(msg)

		// Methods for reading historical values
		case api.ReadEventHistoryMethod:
			return h.HandleReadEventHistory(msg)
		case api.ReadActionHistoryMethod:
			return h.HandleReadActionHistory(msg)
		}
	}
	return nil, nil
}

// HandleReadActionHistory handles the request for a history of actions
func (h *DigiTwinHandler) HandleReadActionHistory(msg *things.ThingMessage) ([]byte, error) {
	//h.svc.History.ReadActionHistory(msg.ThingID, msg.key)
	return nil, fmt.Errorf("not yet implemented")
}

func (h *DigiTwinHandler) HandleReadActions(msg *things.ThingMessage) ([]byte, error) {
	args := api.ReadActionsArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadActions. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadActions(args.ThingID, args.Keys, args.Since)
	if err != nil {
		return nil, err
	}
	resp := api.ReadActionsResp{Messages: values}
	reply, err := json.Marshal(resp)
	return reply, err
}

// HandleReadEventHistory handles the request for a history of events
func (h *DigiTwinHandler) HandleReadEventHistory(msg *things.ThingMessage) ([]byte, error) {
	//h.svc.History.ReadEventHistory(msg.ThingID, msg.key)
	return nil, fmt.Errorf("not yet implemented")
}

func (h *DigiTwinHandler) HandleReadEvents(msg *things.ThingMessage) ([]byte, error) {
	args := api.ReadEventsArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadEvents. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadEvents(args.ThingID, args.Keys, args.Since)
	if err != nil {
		return nil, err
	}
	resp := api.ReadEventsResp{Messages: values}
	reply, err := json.Marshal(resp)
	return reply, err
}

func (h *DigiTwinHandler) HandleReadProperties(msg *things.ThingMessage) ([]byte, error) {
	args := api.ReadPropertiesArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err != nil {
		return nil, fmt.Errorf("HandleReadProperties. Argument error: %w", err)
	}
	values, err := h.svc.Values.ReadProperties(args.ThingID, args.Keys, args.Since)
	if err != nil {
		return nil, err
	}
	resp := api.ReadPropertiesResp{Messages: values}
	reply, err := json.Marshal(resp)
	return reply, err
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

// HandleWriteAction queues or forwards the action to the connected agent for the thing.
func (h *DigiTwinHandler) HandleWriteAction(msg *things.ThingMessage) ([]byte, error) {
	// forward it to the intended destination
	// TODO: queue actions for agents that are configured as such
	// TODO: track action progress status
	//status, err := h.svc.ActionForwarder.WriteAction(args.ThingID, args.Key, args.Value)
	status := api.ActionStatusRejected
	err := fmt.Errorf("not yet implemented")
	if err != nil {
		return nil, err
	}
	reply, err := json.Marshal(status)
	return reply, err
}

// HandleWriteThing handles a request to update a TD by agent
// This is handled the same as a "$td" event by HandleTDEvent
func (h *DigiTwinHandler) HandleWriteThing(msg *things.ThingMessage) ([]byte, error) {
	args := api.WriteThingArgs{}
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = h.svc.Directory.UpdateThing(msg.SenderID, msg.ThingID, args.TD)
	}
	if err != nil {
		slog.Warn("HandleWriteThing: TD update failed", "err", err)
	}
	return nil, err
}

// HandleWriteProperty handles a request to write a property value
// This sends the request to the actual thing or queues it if the thing is not reachable at this point.
func (h *DigiTwinHandler) HandleWriteProperty(msg *things.ThingMessage) ([]byte, error) {
	//var args api.WritePropertyArgs
	//err := json.Unmarshal(msg.Data, &args)
	//if err != nil {
	//	return nil, fmt.Errorf("HandleWriteProperty. Argument error: %w", err)
	//}
	//status, err := h.svc.Values.WriteProperty(args.ThingID, args.Key, args.Value)
	//if err != nil {
	//	return nil, err
	//}
	//reply, err := json.Marshal(status)
	//return reply, err
	return nil, fmt.Errorf("not yet implemented")
}

// NewDigiTwinHandler creates a new instance of the messaging api server for the
// digitwin service.
func NewDigiTwinHandler(svc *service.DigiTwinService) router.MessageHandler {
	decoder := DigiTwinHandler{svc: svc}
	return decoder.HandleMessage
}
