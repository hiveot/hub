// Package service with digital twin action flow handling functions
package router

import (
	"fmt"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"time"
)

// HandleActionResponse handles receiving a response to an action
// This updates the action status if it was recorded.
//
// This:
// 1. Validates the request is still active.
// 2: Updates the status fields of the current digital twin action record to completed.
// 3: Forwards the update to the sender of the request.
// 4: Remove the active request from the cache.
func (svc *DigitwinRouter) HandleActionResponse(resp *transports.ResponseMessage) (err error) {

	// Action response
	svc.requestLogger.Info("<- RESP",
		slog.String("correlationID", resp.CorrelationID),
		slog.String("operation", resp.Operation),
		slog.String("dThingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("status", resp.Status),
		slog.String("Output", resp.ToString(20)),
		slog.String("senderID", resp.SenderID),
	)

	// 1: The response must be an active request
	// Note that event and property subscriptions are active
	svc.mux.Lock()
	as, found := svc.activeCache[resp.CorrelationID]
	svc.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			// FIXME: this happens with writeproperty operations. These should also be in the activeCache
			"HandleResponse: Message '%s' from agent '%s' not in action cache. It is ignored",
			resp.CorrelationID, resp.SenderID)

		svc.requestLogger.Error("Response Failed - correlationID not in action cache",
			slog.String("correlationID", resp.CorrelationID),
		)
		return nil
	}

	// the sender (agents) must be the agent hat handled the action
	if resp.SenderID != as.AgentID {
		err = fmt.Errorf("HandleActionResponse: response ID '%s' of thing '%s' "+
			"does not come from agent '%s' but from '%s'. Response ignored",
			resp.CorrelationID, resp.ThingID, as.AgentID, resp.SenderID)
		svc.requestLogger.Warn(err.Error())
		return nil
	}

	// 2: Update the response status in the digital twin action record and log errors
	// not all requests are tracked.
	_, _ = svc.dtwStore.UpdateActionStatus(resp.SenderID, resp)

	// 3: Forward the response to the sender of the request
	c := svc.transportServer.GetConnectionByConnectionID(as.SenderID, as.ReplyTo)
	if c != nil {
		err = c.SendResponse(resp)
	} else {
		// can't reach the consumer
		err = fmt.Errorf("client connection-id (replyTo) '%s' not found for client '%s'",
			as.ReplyTo, as.SenderID)
	}

	if err != nil {
		svc.requestLogger.Error("Response Failed - Forwarding to sender failed",
			slog.String("correlationID", resp.CorrelationID),
			slog.String("operation", resp.Operation),
			slog.String("dThingID", resp.ThingID),
			slog.String("name", resp.Name),
			slog.String("senderID", resp.SenderID),
			slog.String("status", resp.Status),
			slog.String("err", err.Error()),
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

// HandleSubscriptionNotification handles receiving a subscription update (event, property)
// This updates the digital twin property or event value
func (svc *DigitwinRouter) HandleSubscriptionNotification(resp *transports.ResponseMessage) (err error) {
	// Update the digital twin with this event or property value
	if resp.Operation == wot.OpSubscribeEvent {
		tv := digitwin.ThingValue{
			Name:           resp.Name,
			Output:         resp.Output,
			ThingID:        resp.ThingID,
			Updated:        resp.Updated,
			AffordanceType: transports.AffordanceTypeEvent,
		}
		err = svc.dtwStore.UpdateEventValue(tv)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			svc.transportServer.SendNotification(resp)
		}
	} else if resp.Operation == wot.OpObserveProperty {
		tv := digitwin.ThingValue{
			Name:           resp.Name,
			Output:         resp.Output,
			ThingID:        resp.ThingID,
			Updated:        resp.Updated,
			AffordanceType: transports.AffordanceTypeProperty,
		}
		changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
		// unchanged values are still updated in the store but not published
		// should this be configurable?
		if changed {
			svc.transportServer.SendNotification(resp)
		}
	} else if resp.Operation == wot.OpObserveAllProperties {
		// output is a key-value map
		var propMap map[string]any
		err = tputils.DecodeAsObject(resp.Output, &propMap)
		if err == nil {
			for k, v := range propMap {
				tv := digitwin.ThingValue{
					Name:    k,
					Output:  v,
					ThingID: resp.ThingID,
					Updated: resp.Updated,
				}
				changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
				// unchanged values are still updated in the store but not published
				// should this be configurable?
				if changed {
					svc.transportServer.SendNotification(resp)
				}
			}
		}
	} else if resp.Operation == wot.HTOpUpdateTD {
		tdJSON := resp.ToString(0)
		err := svc.dtwService.DirSvc.UpdateTD(resp.SenderID, tdJSON)
		if err != nil {
			slog.Warn(err.Error())
		}
		// Don't forward the notification.
		//Only digitwin TDs should be published. These have updated forms.
	} else {
		err := fmt.Errorf("Unknown notification '%s'", resp.Operation)
		slog.Warn(err.Error())
		// Other notifications are not supported at this moment
		//svc.cm.PublishNotification(notif)
	}

	return err
}

// HandleResponse update the action status with the agent response.
//
// This converts the ThingID from the agent to that of the digital twin for whom
// the response is intended. The digital twin in turn sends this to the client
// that requested the action on the digital twin.
//
// If the message is no longer in the active cache then it is ignored.
func (svc *DigitwinRouter) HandleResponse(resp *transports.ResponseMessage) error {
	var err error

	// Convert the agent ThingID to that of the digital twin
	dThingID := td.MakeDigiTwinThingID(resp.SenderID, resp.ThingID)
	resp.ThingID = dThingID
	// ensure the updated time is set
	if resp.Updated == "" {
		resp.Updated = time.Now().Format(wot.RFC3339Milli)
	}

	// event/property notifications are forwarded to subscribers
	if resp.Operation == wot.OpObserveProperty ||
		resp.Operation == wot.OpObserveAllProperties ||
		resp.Operation == wot.OpSubscribeEvent ||
		resp.Operation == wot.OpSubscribeAllEvents ||
		resp.Operation == wot.HTOpUpdateTD {

		err = svc.HandleSubscriptionNotification(resp)
		return err
	}
	return svc.HandleActionResponse(resp)
}
