// Package service with digital twin event handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"log/slog"
)

// HandleEventFlow handles the flow of a new event value received from an agent.
//
// This re-submits the event with the digital twin's thing ID in the background,
// and stores it in the digital twin instance. Only the last event value is kept.
func (svc *HubRouter) HandleEventFlow(
	agentID string, thingID string, name string, value any, messageID string) error {
	slog.Info("HandleEventFlow",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("eventName", name),
		slog.String("value", fmt.Sprintf("%v", value)),
	)

	// FIXME: special case: if this is a delivery update event then update the message
	if name == vocab.EventNameDeliveryUpdate {

		// FIXME: locate the action messageID and update its progress

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
