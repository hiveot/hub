package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

// HandleActionRequest passes the action request to the associated Thing.
func (svc *IsyBinding) handleActionRequest(action *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
	if action.Operation == vocab.OpWriteProperty {
		return svc.handleConfigRequest(action)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", action.ThingID),
		slog.String("name", action.Name),
		slog.String("senderID", action.SenderID))

	if !svc.isyAPI.IsConnected() {
		slog.Warn(stat.Error)
		stat.Failed(action, fmt.Errorf("No connection with the gateway"))
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
	// publish any changes that are the result of the action
	go func() {
		values := isyThing.GetPropValues(true)
		if len(values) > 0 {
			_ = svc.hc.PubMultipleProperties(isyThing.GetID(), values)
		}
	}()
	return stat
}
