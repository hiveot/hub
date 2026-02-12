package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/messaging"
)

// HandleRequest passes the action request to the associated Thing.
func (svc *IsyBinding) handleRequest(req *messaging.RequestMessage,
	_ messaging.IConnection) (resp *messaging.ResponseMessage) {

	if req.Operation == vocab.OpWriteProperty {
		return svc.handleConfigRequest(req)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	if !svc.isyAPI.IsConnected() {
		resp = req.CreateResponse(nil, fmt.Errorf("No connection with the gateway"))
		slog.Warn(resp.Error.String())
		return
	}
	isyThing := svc.IsyGW.GetIsyThing(req.ThingID)
	if isyThing == nil {
		err := fmt.Errorf("handleActionRequest: thing '%s' not found", req.ThingID)
		resp = req.CreateResponse(nil, err)
		slog.Warn(resp.Error.String())
		return
	}
	// FIXME-1: how to determine the output of an action with ISY?
	// FIXME-2: how to determine an action has completed?
	resp = isyThing.HandleActionRequest(svc.ag, req)

	// publish any changes that are the result of the action
	go func() {
		thingID := req.ThingID
		values := isyThing.GetPropValues(true)
		for k, v := range values {
			_ = svc.ag.PubProperty(thingID, k, v)
		}
	}()
	return resp
}
