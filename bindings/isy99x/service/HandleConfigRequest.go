package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/transports"
	"log/slog"
)

// handleConfigRequest for handling device configuration changes
func (svc *IsyBinding) handleConfigRequest(req *transports.RequestMessage) (resp *transports.ResponseMessage) {

	slog.Info("handleConfigRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	// configuring the binding doesn't require a connection with the gateway
	if req.ThingID == svc.thingID {
		resp = svc.HandleWriteBindingProperty(req)
		return
	}

	if !svc.isyAPI.IsConnected() {
		// this is a delivery failure
		resp = req.CreateResponse(nil, fmt.Errorf("no connection with the gateway"))
		slog.Warn(resp.Error)
		return resp
	}

	// pass request to the Thing
	isyThing := svc.IsyGW.GetIsyThing(req.ThingID)
	if isyThing == nil {
		resp = req.CreateResponse(nil, fmt.Errorf("handleActionRequest: thing '%s' not found", req.ThingID))
		slog.Warn(resp.Error)
		return
	}
	resp = isyThing.HandleConfigRequest(req)

	// publish changed values after returning
	go func() {
		values := isyThing.GetPropValues(true)
		_ = svc.ag.PubProperties(isyThing.GetID(), values)

		// re-submit the TD if the title changes
		if req.Name == vocab.PropDeviceTitle {
			td := isyThing.MakeTD()
			_ = svc.ag.PubTD(td)
		}
	}()
	return resp
}
