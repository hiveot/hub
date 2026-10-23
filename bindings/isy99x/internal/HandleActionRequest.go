package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
)

// HandleRequest passes the action request to the associated Thing.
// This distributes the requests to the thing identified in the thingID
func (svc *IsyBinding) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.Operation == td.OpWriteProperty {
		return svc.handleConfigRequest(req, replyTo)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	if !svc.isyAPI.IsConnected() {
		err := fmt.Errorf("No connection with the gateway")
		slog.Warn(err.Error())
		return err
	}
	isyThing := svc.IsyGW.GetIsyThing(req.ThingID)
	if isyThing == nil {
		err := fmt.Errorf("handleActionRequest: thing '%s' not found", req.ThingID)
		slog.Warn(err.Error())
		return err
	}
	// FIXME-1: how to determine the output of an action with ISY?
	// FIXME-2: how to determine an action has completed?
	err := isyThing.HandleActionRequest(req, replyTo)

	// publish any changes that are the result of the action
	go func() {
		thingID := req.ThingID
		values := isyThing.GetPropValues(true)
		for k, v := range values {
			svc.PubProperty(thingID, k, v, true)
		}
	}()
	return resp
}
