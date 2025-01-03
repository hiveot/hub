// Package service with digital twin action flow handling functions
package router

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"time"
)

// HandleResponse update the action status with the agent response.
//
// This converts the ThingID from the agent to that of the digital twin for whom
// the response is intended. The digital twin in turn sends this to the client
// that requested the action on the digital twin.
//
// This updates the action status if it was recorded.
//
// If the response status is StatusCompleted then the 'output' contains the
// action result, as described in the TD action affordance.
//
// This:
// 1. Validates the request is still active.
// 2: Updates the status fields of the current digital twin action record to completed.
// 3: Forwards the update to the sender of the request.
// 4: Remove the active request from the cache.
//
// If the message is no longer in the active cache then it is ignored.
func (svc *DigitwinRouter) HandleResponse(resp transports.ResponseMessage) error {
	var err error

	// Convert the agent ThingID to that of the digital twin
	dThingID := td.MakeDigiTwinThingID(resp.SenderID, resp.ThingID)
	resp.ThingID = dThingID
	// ensure the updated time is set
	if resp.Updated == "" {
		resp.Updated = time.Now().Format(wot.RFC3339Milli)
	}

	if resp.CorrelationID == "" {
		slog.Warn("Received a response without a correlationID. This is ignored")
		return nil
	}

	slog.Info("HandleResponse",
		slog.String("ThingID", resp.ThingID),
		slog.String("Name", resp.Name),
		slog.String("CorrelationID", resp.CorrelationID),
	)

	// 1: The response must be an active request
	svc.mux.Lock()
	as, found := svc.activeCache[resp.CorrelationID]
	svc.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			"HandleResponse: Message '%s' from agent '%s' not in action cache. It is ignored",
			resp.CorrelationID, resp.SenderID)
		slog.Warn(err.Error())
		return nil
	}

	// the sender (agents) must be the agent hat handled the action
	if resp.SenderID != as.AgentID {
		err = fmt.Errorf("HandleActionResponse: response ID '%s' of thing '%s' "+
			"does not come from agent '%s' but from '%s'. Response ignored",
			resp.CorrelationID, resp.ThingID, as.AgentID, resp.SenderID)
		slog.Warn(err.Error())
		return nil
	}

	// 2: Update the response status in the digital twin action record and log errors
	// not all requests are tracked.
	_, _ = svc.dtwStore.UpdateActionStatus(resp.SenderID, resp)

	// 3: Forward the response to the sender of the request
	c := svc.cm.GetConnectionByConnectionID(as.ReplyTo)
	if c != nil {
		err = c.SendResponse(resp)
	} else {
		// can't reach the consumer
		err = fmt.Errorf("client connection-id (replyTo) '%s' not found for client '%s'",
			as.ReplyTo, as.SenderID)
	}

	if err != nil {
		slog.Warn("HandleActionResponse. Forwarding to sender failed",
			slog.String("senderID", as.SenderID),
			slog.String("thingID", as.ThingID),
			slog.String("replyTo", as.ReplyTo),
			slog.String("err", err.Error()),
			slog.String("CorrelationID", as.CorrelationID),
		)
		err = nil
	}

	// 4: Update the active action cache and remove the action when completed or failed
	if resp.Status == transports.StatusCompleted || resp.Status == transports.StatusFailed {
		svc.mux.Lock()
		defer svc.mux.Unlock()
		delete(svc.activeCache, as.CorrelationID)
	}
	return nil
}
