package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	jsoniter "github.com/json-iterator/go"
)

// handleConfigRequest for handling device configuration changes
func (svc *IsyBinding) handleConfigRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	slog.Info("handleConfigRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	// configuring the binding doesn't require a connection with the gateway
	if req.ThingID == svc.thingID {
		resp := svc.HandleWriteBindingProperty(req)
		return replyTo(resp)
	}

	if !svc.isyAPI.IsConnected() {
		// this is a delivery failure
		resp := req.CreateResponse(nil, fmt.Errorf("no connection with the gateway"))
		slog.Warn(resp.Error.String())
		return replyTo(resp)
	}

	// pass request to the Thing
	isyThing := svc.IsyGW.GetIsyThing(req.ThingID)
	if isyThing == nil {
		resp := req.CreateResponse(nil, fmt.Errorf("handleConfigRequest: thing '%s' not found", req.ThingID))
		slog.Warn(resp.Error.String())
		return replyTo(resp)
	}
	resp := isyThing.HandleConfigRequest(req)

	// publish changed values after returning
	go func() {
		values := isyThing.GetPropValues(true)
		svc.PubProperties(isyThing.GetID(), values, true)

		// re-submit the TD if the title changes
		if req.Name == td.WoTTitle {
			tdi := isyThing.MakeTD()
			tdJSON, _ := jsoniter.MarshalToString(tdi)
			svc.WriteTD(tdJSON)
			//_ = svc.ag.UpdateThing(tdi)
		}
	}()
	return replyTo(resp)
}
