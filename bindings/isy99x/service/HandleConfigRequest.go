package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"log/slog"
)

// handleConfigRequest for handling binding, gateway and node configuration changes
func (svc *IsyBinding) handleConfigRequest(action *transports.ThingMessage) (stat transports.RequestStatus) {

	slog.Info("handleConfigRequest",
		slog.String("thingID", action.ThingID),
		slog.String("name", action.Name),
		slog.String("senderID", action.SenderID))

	// configuring the binding doesn't require a connection with the gateway
	if action.ThingID == svc.thingID {
		err := svc.HandleBindingConfig(action)
		stat.Completed(action, nil, err)
		return
	}

	if !svc.isyAPI.IsConnected() {
		// this is a delivery failure
		stat.Failed(action, fmt.Errorf("no connection with the gateway"))
		slog.Warn(stat.Error)
		return
	}

	// pass request to the Thing
	isyThing := svc.IsyGW.GetIsyThing(action.ThingID)
	if isyThing == nil {
		stat.Failed(action, fmt.Errorf("handleActionRequest: thing '%s' not found", action.ThingID))
		slog.Warn(stat.Error)
		return
	}
	err := isyThing.HandleConfigRequest(action)
	stat.Completed(action, nil, err)

	// publish changed values after returning
	go func() {
		values := isyThing.GetPropValues(true)
		_ = svc.hc.PubMultipleProperties(isyThing.GetID(), values)

		// re-submit the TD if the title changes
		if action.Name == vocab.PropDeviceTitle {
			td := isyThing.MakeTD()
			tdJSON, _ := json.Marshal(td)

			_ = svc.hc.PubTD(td.ID, string(tdJSON))
		}
	}()
	return stat
}
