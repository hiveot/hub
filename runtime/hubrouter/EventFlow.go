// Package service with digital twin event handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
)

// HandleEventFlow handles the flow of a new event value received from an agent.
//
// This re-submits the event with the digital twin's thing ID in the background,
// and stores it in the digital twin instance. Only the last event value is kept.
//
// Two special events are handled here: $td with a TD updated and $delivery with
// a delivery update.
func (svc *HubRouter) HandleEventFlow(
	agentID string, thingID string, name string, value any, messageID string) error {
	slog.Info("HandleEventFlow",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("eventName", name),
		slog.String("value", fmt.Sprintf("%v", value)),
	)

	// FIXME: special case: action flow handles delivery updates
	// FIXME: This is non-WoT compatible. Sender MUST use a DeliveryStatus payload
	if name == vocab.EventNameDeliveryUpdate {
		// forward the delivery status to the sender (consumer)
		// expect a delivery status message
		stat := hubclient.DeliveryStatus{}
		err := utils.DecodeAsObject(value, &stat)
		if err != nil {
			slog.Warn("Decoding status update failed:", "err", err.Error)
		}
		svc.HandleDeliveryUpdate(agentID, stat)
	} else if name == vocab.EventNameTD {
		// update the TD: this is a legacy event. Looking for a beter way.
		dtdJSON := utils.DecodeAsString(value)
		err := svc.dtwService.DirSvc.UpdateDTD(agentID, dtdJSON)
		if err != nil {
			slog.Warn("Updating TD failed:", "err", err.Error())
		}
	} else {

		dThingID, err := svc.dtwStore.UpdateEventValue(agentID, thingID, name, value, messageID)
		if err != nil {
			return err
		}
		if svc.tb == nil {
			return fmt.Errorf("HandleEventFlow: no transport binding")
		}
		// resubmit the event to subscribers of the digital twin in the background
		go svc.tb.PublishEvent(dThingID, name, value, messageID)
	}
	return nil
}
