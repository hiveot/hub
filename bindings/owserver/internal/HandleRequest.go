package internal

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hub/bindings/owserver"
)

const MaxUpdateWaitTime = 3

// HandleRequest handles action and property requests for Things managed
// by the OWServer gateway.
//
// For 1-wire, configuration and actions are the same thing
func (svc *OWServerGateway) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	slog.Info("HandleRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("payload", req.ToString(20)),
	)
	// Custom config. Change device title for any node.
	if req.Operation == wot.OpWriteProperty {
		if req.Name == wot.WoTTitle {
			err := svc.SetTitle(req)
			deviceTitleByte, _ := svc.customTitles.Get(req.ThingID)
			resp := req.CreateResponse(string(deviceTitleByte), err)
			return replyTo(resp)
		}
	}
	if req.ThingID == svc.GetID() || req.ThingID == owserver.OWServerCellType {
		// this is a property update for the gateway service itself
		var err error

		// Custom config. Change device title.
		// let the base handle read operations
		err = svc.ExposedThing.HandleRequest(req, replyTo)
		return err
	} else {
		// pass it on to the node
		owThing, _ := svc.GetOWThing(req.ThingID)
		if owThing == nil {
			err := fmt.Errorf("HandleRequest: Thing '%s' is not a known 1-wire device", req.ThingID)
			slog.Warn(err.Error())
			return err
		}
		return owThing.HandleRequest(req, replyTo)
	}
}

// SetTitle configures a new title of a thing
//
// OWServer doesn't support this so store this in the service store.
func (svc *OWServerGateway) SetTitle(req *msg.RequestMessage) (err error) {

	valueStr := req.ToString(0)

	err = svc.customTitles.Set(req.ThingID, []byte(valueStr))
	if err != nil {
		slog.Error("HandleConfigRequest: Unable to save title", "err", err.Error())
	}
	if err == nil {
		// publish notification with the new title after returning
		go svc.PubThingProperty(req.ThingID, wot.WoTTitle, valueStr)
	}
	return err
}

// WaitForValueUpdate waits for the node value to be applied
func (svc *OWServerGateway) WaitForValueUpdate(
	req *msg.RequestMessage, seconds int) (newValue string, hasUpdated bool) {

	var err error
	valueStr := req.ToString(0)
	hasUpdated = false

	for i := 0; i < seconds; i++ {
		newValue, err = svc.edsAPI.ReadNodeValue(req.ThingID, req.Name)
		if err != nil {
			slog.Error("WaitForValueUpdate: error", "err", err.Error())
			hasUpdated = false
			break
		}
		if valueStr == newValue {
			hasUpdated = true
			break
		}
		time.Sleep(time.Second * 1)
	}
	return newValue, hasUpdated
}
