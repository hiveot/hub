// Package hubrouter with digital twin event routing functions
package router

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// HandleNotification routes notifications from agents to clients
// This updates the last event/property value in the digital twin store.
func (svc *DigitwinRouter) HandleNotification(agentID string, notif transports.NotificationMessage) {
	// convert the ThingID to that of the digital twin
	dThingID := td.MakeDigiTwinThingID(agentID, notif.ThingID)
	notif.ThingID = dThingID

	if notif.Operation == wot.HTOpEvent {

		err := svc.dtwStore.UpdateEventValue(dThingID, notif.Name, notif.Data)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			go svc.cm.PublishNotification(notif)
		}
	} else if notif.Operation == wot.HTOpUpdateProperty {

		changed, _ := svc.dtwStore.UpdatePropertyValue(
			notif.ThingID, notif.Name, notif.Data, "")
		if changed {
			svc.cm.PublishNotification(notif)
		}

	} else if notif.Operation == wot.HTOpUpdateTD {
		tdJSON := notif.ToString()
		svc.dtwService.DirSvc.UpdateTD(agentID, tdJSON)
		svc.cm.PublishNotification(notif)

	} else {
		err := fmt.Errorf("Unknown notification '%s'", notif.Operation)
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

// HandleUpdateMultipleProperties agent publishes a batch with multiple property values.
// This sends individual property updates to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propMap map of property key-values
func (svc *DigitwinRouter) HandleUpdateMultipleProperties(msg transports.NotificationMessage) {
	propMap := make(map[string]any)
	err := tputils.Decode(msg.Data, &propMap)
	if err != nil {
		slog.Warn("HandleUpdateMultipleProperties: error decoding property map", "err", err.Error())
		return
	}
	// update the property in the digitwin and notify observers for each change
	changes, err := svc.dtwStore.UpdateProperties(msg.ThingID, propMap, "")
	if len(changes) > 0 {
		for k, v := range changes {
			notif := transports.NewNotificationMessage(wot.HTOpUpdateProperty, msg.ThingID, k, v)
			svc.cm.PublishNotification(notif)
		}
	}
}
