// Package router with digital twin event routing functions
package router

import (
	"fmt"
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
	dThingID := td.MakeDigiTwinThingID(notif.SenderID, notif.ThingID)
	notif.ThingID = dThingID
	if notif.Timestamp == "" {
		notif.Timestamp = time.Now().Format(wot.RFC3339Milli)
	}

	// Update the digital twin with this event or property value
	if notif.Operation == wot.OpSubscribeEvent ||
		notif.Operation == wot.OpSubscribeAllEvents {
		tv := digitwin.ThingValue{
			Name:           notif.Name,
			Output:         notif.Data,
			ThingID:        notif.ThingID,
			Updated:        notif.Timestamp,
			AffordanceType: messaging.AffordanceTypeEvent,
		}
		err = svc.dtwStore.UpdateEventValue(tv)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			svc.transportServer.SendNotification(notif)
		}
	} else if notif.Operation == wot.OpObserveProperty {
		tv := digitwin.ThingValue{
			Name:           notif.Name,
			Output:         notif.Data,
			ThingID:        notif.ThingID,
			Updated:        notif.Timestamp,
			AffordanceType: messaging.AffordanceTypeProperty,
		}
		changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
		// unchanged values are still updated in the store but not published
		// should this be configurable?
		if changed {
			svc.transportServer.SendNotification(notif)
		}
	} else if notif.Operation == wot.OpObserveAllProperties {
		// output is a key-value map
		var propMap map[string]any
		err := tputils.DecodeAsObject(notif.Data, &propMap)
		if err == nil {
			for k, v := range propMap {
				tv := digitwin.ThingValue{
					Name:    k,
					Output:  v,
					ThingID: notif.ThingID,
					Updated: notif.Timestamp,
				}
				changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
				// unchanged values are still updated in the store but not published
				// should this be configurable?
				if changed {
					// notify the consumer with individual updates instead of a map
					// this seems more correct than sending a map.
					notif := *notif
					notif.Operation = wot.OpObserveProperty
					notif.Name = k
					notif.Data = v
					svc.transportServer.SendNotification(&notif)
				}
			}
		}
		//} else if notif.Operation == wot.HTOpUpdateTD {
		//	tdJSON := notif.ToString(0)
		//	err := svc.dtwService.DirSvc.UpdateTD(notif.SenderID, tdJSON)
		//	if err != nil {
		//		slog.Warn(err.Error())
		//	}
	} else {
		err = fmt.Errorf("Unknown notification '%s'", notif.Operation)
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
