// Package router with digital twin event routing functions
package router

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"time"
)

// HandleNotification handles receiving a notification from an agent (event, property, action)
// This updates the digital twin property or event value
func (svc *DigitwinRouter) HandleNotification(notif *messaging.NotificationMessage) {
	var err error
	svc.notifLogger.Info("<- NOTIF: HandleNotification",
		slog.String("operation", notif.Operation),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
		slog.String("value", tputils.DecodeAsString(notif.Data, 30)),
	)
	// Convert the agent ThingID to that of the digital twin
	notifCpy := *notif
	dThingID := td.MakeDigiTwinThingID(notif.SenderID, notif.ThingID)
	notifCpy.ThingID = dThingID
	if notifCpy.Timestamp == "" {
		notifCpy.Timestamp = utils.FormatUTCMilli(time.Now())
	}

	// Update the digital twin with this event or property value
	if notifCpy.Operation == wot.OpSubscribeEvent ||
		notifCpy.Operation == wot.OpSubscribeAllEvents {
		tv := digitwin.ThingValue{
			Name:           notifCpy.Name,
			Output:         notifCpy.Data,
			ThingID:        notifCpy.ThingID,
			Updated:        notifCpy.Timestamp,
			AffordanceType: messaging.AffordanceTypeEvent,
		}
		err = svc.dtwStore.UpdateEventValue(tv)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			svc.transportServer.SendNotification(&notifCpy)
		}
	} else if notifCpy.Operation == wot.OpObserveProperty {
		tv := digitwin.ThingValue{
			Name:           notifCpy.Name,
			Output:         notifCpy.Data,
			ThingID:        notifCpy.ThingID,
			Updated:        notifCpy.Timestamp,
			AffordanceType: messaging.AffordanceTypeProperty,
		}
		changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
		// unchanged values are still updated in the store but not published
		// should this be configurable?
		if changed {
			svc.transportServer.SendNotification(&notifCpy)
		}
	} else if notifCpy.Operation == wot.OpObserveAllProperties {
		// output is a key-value map
		var propMap map[string]any
		err := tputils.DecodeAsObject(notifCpy.Data, &propMap)
		if err == nil {
			for k, v := range propMap {
				tv := digitwin.ThingValue{
					Name:    k,
					Output:  v,
					ThingID: notifCpy.ThingID,
					Updated: notifCpy.Timestamp,
				}
				changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
				// unchanged values are still updated in the store but not published
				// should this be configurable?
				if changed {
					// notify the consumer with individual updates instead of a map
					// this seems more correct than sending a map.
					notifCpy := *notif
					notifCpy.Operation = wot.OpObserveProperty
					notifCpy.Name = k
					notifCpy.Data = v
					svc.transportServer.SendNotification(&notifCpy)
				}
			}
		}
	} else if notifCpy.Operation == wot.OpInvokeAction {
		// action progress update. Forward to sender of the request
		var cc messaging.IConnection
		svc.mux.Lock()
		actRec, found := svc.activeCache[notifCpy.CorrelationID]
		svc.mux.Unlock()
		if !found {
			// no associated action for this notification
			return
		}
		// the sender (agents) must be the agent hat handled the action
		if notifCpy.SenderID != actRec.AgentID {
			// notification wasn't sent by the correct sender
			slog.Warn("Notification with correlationID '%s' from sender '%s' is "+
				"not sent by agent '%s' for action '%s'",
				notifCpy.CorrelationID, notifCpy.SenderID, actRec.AgentID, notifCpy.Operation,
			)
			return
		}
		// update the action status
		svc.dtwStore.UpdateActionWithNotification(&notifCpy)

		// forward the notification to the sender of the request only
		cc = svc.transportServer.GetConnectionByConnectionID(actRec.SenderID, actRec.ReplyTo)
		if cc != nil {
			_ = cc.SendNotification(&notifCpy)
		}
	} else {
		err = fmt.Errorf("Unknown notification '%s'", notifCpy.Operation)
	}
	if err != nil {
		slog.Warn(err.Error())
		// Other notifications are not supported at this moment
		//svc.cm.PublishNotification(notif)
	}
}

//// HandleUpdateTD agent notifies of a TD refresh.
//// This converts the operation in an action for the directory service.
//func (svc *DigitwinRouter) HandleUpdateTD(msg transports.NotificationMessage) {
//
//	msg.ThingID = digitwin.DirectoryDThingID
//	msg.Name = digitwin.DirectoryUpdateTDMethod
//	output, err = svc.digitwinAction(&dirMsg)
//	return true, output, err
//}
