package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// HandleActionRequest passes the action request to the associated Thing.
func (svc *IsyBinding) handleActionRequest(action *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	if action.MessageType == vocab.MessageTypeProperty {
		return svc.handleConfigRequest(action)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", action.ThingID),
		slog.String("key", action.Key),
		slog.String("senderID", action.SenderID))

	if !svc.isyAPI.IsConnected() {
		slog.Warn(stat.Error)
		stat.Completed(action, nil, fmt.Errorf("No connection with the gateway"))
		return
	}
	isyThing := svc.IsyGW.GetIsyThing(action.ThingID)
	if isyThing == nil {
		stat.Completed(action, nil, fmt.Errorf("handleActionRequest: thing '%s' not found", action.ThingID))
		slog.Warn(stat.Error)
		return
	}
	err := isyThing.HandleActionRequest(action)
	stat.Completed(action, nil, err)
	return
}
