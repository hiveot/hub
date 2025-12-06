package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/wot"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
)

// handleConfigRequest for handling device configuration changes
func (svc *IsyBinding) handleConfigRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {

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
		slog.Warn(resp.Error.String())
		return resp
	}

	// pass request to the Thing
	isyThing := svc.IsyGW.GetIsyThing(req.ThingID)
	if isyThing == nil {
		resp = req.CreateResponse(nil, fmt.Errorf("handleConfigRequest: thing '%s' not found", req.ThingID))
		slog.Warn(resp.Error.String())
		return
	}
	resp = isyThing.HandleConfigRequest(req)

	// publish changed values after returning
	go func() {
		values := isyThing.GetPropValues(true)
		_ = svc.ag.PubProperties(isyThing.GetID(), values)

		// re-submit the TD if the title changes
		if req.Name == wot.WoTTitle {
			tdi := isyThing.MakeTD()
			tdJSON, _ := jsoniter.MarshalToString(tdi)
			_ = digitwin.ThingDirectoryUpdateThing(svc.ag.Consumer, tdJSON)
			//_ = svc.ag.UpdateThing(tdi)
		}
	}()
	return resp
}
