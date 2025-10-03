// Package router with digital twin event routing functions
package router

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
)

// HandleNotification handles receiving a notification from an agent (event, property, action)
// This updates the digital twin property or event value
func (r *DigitwinRouter) HandleNotification(notif *messaging.NotificationMessage) {
	var err error
	r.notifLogger.Info("<- NOTIF: HandleNotification",
		slog.String("senderID", notif.SenderID),
		slog.String("operation", notif.Operation),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
		slog.String("value", tputils.DecodeAsString(notif.Value, 30)),
	)
	// Convert the agent ThingID to that of the digital twin
	dtwNotif := *notif
	dThingID := td.MakeDigiTwinThingID(notif.SenderID, notif.ThingID)
	dtwNotif.ThingID = dThingID
	if dtwNotif.Timestamp == "" {
		dtwNotif.Timestamp = utils.FormatUTCMilli(time.Now())
	}

	// Update the digital twin with this event or property value
	if dtwNotif.Operation == wot.OpSubscribeEvent ||
		dtwNotif.Operation == wot.OpSubscribeAllEvents {
		tv := digitwin.ThingValue{
			Name:           dtwNotif.Name,
			Data:           dtwNotif.Value,
			ThingID:        dtwNotif.ThingID,
			Timestamp:      dtwNotif.Timestamp,
			AffordanceType: string(messaging.AffordanceTypeEvent),
		}
		err = r.dtwStore.UpdateEventValue(tv)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			r.transportServer.SendNotification(&dtwNotif)
		}
	} else if dtwNotif.Operation == wot.OpObserveProperty {
		tv := digitwin.ThingValue{
			Name:           dtwNotif.Name,
			Data:           dtwNotif.Value,
			ThingID:        dtwNotif.ThingID,
			Timestamp:      dtwNotif.Timestamp,
			AffordanceType: string(messaging.AffordanceTypeProperty),
		}
		changed, _ := r.dtwStore.UpdatePropertyValue(tv)
		// unchanged values are still updated in the store but not published
		// should this be configurable?
		if changed {
			r.transportServer.SendNotification(&dtwNotif)
		}
	} else if dtwNotif.Operation == wot.OpObserveAllProperties {
		// output is a key-value map
		var propMap map[string]any
		err := tputils.DecodeAsObject(dtwNotif.Value, &propMap)
		if err == nil {
			for k, v := range propMap {
				tv := digitwin.ThingValue{
					AffordanceType: string(messaging.AffordanceTypeProperty),
					Name:           k,
					Data:           v,
					ThingID:        dtwNotif.ThingID,
					Timestamp:      dtwNotif.Timestamp,
				}
				changed, _ := r.dtwStore.UpdatePropertyValue(tv)
				// unchanged values are still updated in the store but not published
				// should this be configurable?
				if changed {
					// notify the consumer with individual updates instead of a map
					// this seems more correct than sending a map.
					notifCpy2 := dtwNotif
					notifCpy2.Operation = wot.OpObserveProperty
					notifCpy2.Name = k
					notifCpy2.Value = v
					r.transportServer.SendNotification(&notifCpy2)
				}
			}
		}
	} else if dtwNotif.Operation == wot.OpInvokeAction {
		// action progress update. Forward to sender of the request
		var cc messaging.IConnection
		r.mux.Lock()
		actRec, found := r.activeCache[dtwNotif.CorrelationID]
		r.mux.Unlock()
		if !found {
			// no associated action for this notification
			return
		}
		// the sender (agents) must be the agent hat handled the action
		if dtwNotif.SenderID != actRec.AgentID {
			// notification wasn't sent by the correct sender
			slog.Warn("Notification with correlationID '%s' from sender '%s' is "+
				"not sent by agent '%s' for action '%s'",
				dtwNotif.CorrelationID, dtwNotif.SenderID, actRec.AgentID, dtwNotif.Operation,
			)
			return
		}
		// update the action status
		r.dtwStore.UpdateActionWithNotification(&dtwNotif)

		// forward the notification to the sender of the request only
		cc = r.transportServer.GetConnectionByConnectionID(actRec.SenderID, actRec.ReplyTo)
		if cc != nil {
			_ = cc.SendNotification(&dtwNotif)
		}
	} else {
		err = fmt.Errorf("Unknown notification '%s'", dtwNotif.Operation)
	}
	if err != nil {
		slog.Warn(err.Error())
		// Other notifications are not supported at this moment
		//svc.cm.PublishNotification(notif)
	}
}
