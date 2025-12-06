// Package service with digital twin action flow handling functions
package router

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
)

// HandleActionResponse handles receiving a response to an action
// This updates the action status if it was recorded.
//
// This:
// 1. Validates the request is still active.
// 2: Updates the status fields of the current digital twin action record to completed.
// 3: Forwards the update to the sender of the request.
// 4: Remove the active request from the cache.
func (r *DigitwinRouter) HandleActionResponse(resp *messaging.ResponseMessage) (err error) {

	// Action response
	errMsg := ""
	if resp.Error != nil {
		errMsg = resp.Error.String()
	}
	// log the response
	r.requestLogger.Info("<- RESP",
		slog.String("correlationID", resp.CorrelationID),
		slog.String("operation", resp.Operation),
		slog.String("dThingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("error", errMsg),
		slog.String("Value", resp.ToString(20)),
		slog.String("senderID", resp.SenderID),
	)

	// 1: The response must be an active request
	// Note that event and property subscriptions are active
	r.mux.Lock()
	as, found := r.activeCache[resp.CorrelationID]
	r.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			// FIXME: this happens with writeproperty operations. These should also be in the activeCache
			"HandleResponse: Message '%s' from agent '%s' not in action cache. It is ignored",
			resp.CorrelationID, resp.SenderID)

		r.requestLogger.Error("Response Failed - correlationID not in action cache",
			slog.String("correlationID", resp.CorrelationID),
		)
		return nil
	}

	// the sender (agents) must be the agent hat handled the action
	if resp.SenderID != as.AgentID {
		err = fmt.Errorf("HandleActionResponse: response ID '%s' of thing '%s' "+
			"does not come from agent '%s' but from '%s'. Response ignored",
			resp.CorrelationID, resp.ThingID, as.AgentID, resp.SenderID)
		r.requestLogger.Warn(err.Error())
		return nil
	}

	// 2: Update the response status in the digital twin action record and log errors
	// only action requests are tracked.
	_, _ = r.dtwStore.UpdateActionWithResponse(resp)

	// 3: Forward the response to the sender of the request
	c := r.transportServer.GetConnectionByConnectionID(as.SenderID, as.ReplyTo)
	if c != nil {
		err = c.SendResponse(resp)
	} else {
		// can't reach the consumer
		err = fmt.Errorf("client connection-id (replyTo) '%s' not found for client '%s'",
			as.ReplyTo, as.SenderID)
	}

	if err != nil {
		r.requestLogger.Error("Response Failed - Forwarding to sender failed",
			slog.String("correlationID", resp.CorrelationID),
			slog.String("operation", resp.Operation),
			slog.String("dThingID", resp.ThingID),
			slog.String("name", resp.Name),
			slog.String("senderID", resp.SenderID),
			slog.String("err", err.Error()),
		)
		err = nil
	}

	// 4: Remove the action when the response is received
	//  (notifications can provide intermediate status updates)
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.activeCache, as.CorrelationID)
	return err
}

// HandleResponse update the action status with the agent response.
//
// This converts the ThingID from the agent to that of the digital twin for whom
// the response is intended. The digital twin in turn sends this to the client
// that requested the action on the digital twin.
//
// If the message is no longer in the active cache then it is ignored.
func (r *DigitwinRouter) HandleResponse(resp *messaging.ResponseMessage) error {
	var err error

	// Convert the agent ThingID to that of the digital twin
	dThingID := td.MakeDigiTwinThingID(resp.SenderID, resp.ThingID)
	resp.ThingID = dThingID
	// ensure the updated time is set
	if resp.Timestamp == "" {
		resp.Timestamp = utils.FormatNowUTCMilli()
	}

	// for now the only external response message is an action response
	// (how about write property?)
	err = r.HandleActionResponse(resp)
	return err
}
