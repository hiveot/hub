// Package hubrouter with digital twin event routing functions
package router

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"time"
)

// HandleNotification routes notifications from agents to clients
// This updates the last event/property value in the digital twin store.
func (svc *DigitwinRouter) HandleNotification(notif transports.NotificationMessage) {

	// convert the ThingID to that of the digital twin
	// ensure a 'created' time is set
	dThingID := td.MakeDigiTwinThingID(notif.SenderID, notif.ThingID)
	notif.ThingID = dThingID
	if notif.Created == "" {
		notif.Created = time.Now().Format(wot.RFC3339Milli)
	}

	slog.Info("HandleNotification",
		slog.String("operation", notif.Operation),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
		// this is temporary
		slog.String("data", notif.ToString(20)),
		slog.String("created", notif.Created),
	)

	if notif.Operation == wot.HTOpEvent {
		tv := digitwin.ThingValue{
			Created: notif.Created,
			Data:    notif.Data,
			Name:    notif.Name,
			ThingID: notif.ThingID,
		}
		err := svc.dtwStore.UpdateEventValue(tv)
		if err == nil {
			// broadcast the event to subscribers of the digital twin
			go svc.cm.PublishNotification(notif)
		}
	} else if notif.Operation == wot.HTOpUpdateProperty {

		tv := digitwin.ThingValue{
			Created: notif.Created,
			Data:    notif.Data,
			Name:    notif.Name,
			ThingID: notif.ThingID,
		}
		changed, _ := svc.dtwStore.UpdatePropertyValue(tv)
		if changed {
			svc.cm.PublishNotification(notif)
		}

	} else if notif.Operation == wot.HTOpUpdateMultipleProperties {
		svc.HandleUpdateMultipleProperties(notif)

	} else if notif.Operation == wot.HTOpUpdateTD {
		tdJSON := notif.ToString(0)
		err := svc.dtwService.DirSvc.UpdateTD(notif.SenderID, tdJSON)
		if err != nil {
			slog.Warn(err.Error())
		}
		// Don't forward the notification.
		//Only digitwin TDs should be published. These have updated forms.

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

// HandleUpdateMultipleProperties agent publishes a batch with multiple property
// key-value pairs.
// This sends individual property updates to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// propMap map of property key-values
func (svc *DigitwinRouter) HandleUpdateMultipleProperties(notif transports.NotificationMessage) {
	propMap := make(map[string]any)
	err := tputils.Decode(notif.Data, &propMap)
	if err != nil {
		slog.Warn("HandleUpdateMultipleProperties: error decoding property map",
			"op", notif.Operation,
			"clientID", notif.SenderID,
			"thingID", notif.ThingID,
			"err", err.Error())
		return
	}
	// update the property in the digitwin and notify observers for each change
	changes, err := svc.dtwStore.UpdateProperties(notif.ThingID, notif.Created, propMap)
	if len(changes) > 0 {
		for k, v := range changes {
			notif := transports.NewNotificationMessage(wot.HTOpUpdateProperty, notif.ThingID, k, v)
			svc.cm.PublishNotification(notif)
		}
	}
}
