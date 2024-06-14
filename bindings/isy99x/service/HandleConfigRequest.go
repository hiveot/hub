package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// handleConfigRequest for handling binding, gateway and node configuration changes
func (svc *IsyBinding) handleConfigRequest(action *things.ThingMessage) (stat hubclient.DeliveryStatus) {

	slog.Info("handleConfigRequest",
		slog.String("thingID", action.ThingID),
		slog.String("key", action.Key),
		slog.String("senderID", action.SenderID))

	// configuring the binding doesn't require a connection with the gateway
	if action.ThingID == svc.thingID {
		err := svc.HandleBindingConfig(action)
		stat.Completed(action, err)
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
	stat.Completed(action, err)

	// publish changed values after returning
	go func() {
		_ = svc.PublishNodeValues(true)
		// re-submit the TD if the title changes
		if action.Key == vocab.PropDeviceTitle {
			td := isyThing.GetTD()
			_ = svc.hc.PubTD(td)
		}
	}()
	return stat
}
