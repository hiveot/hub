// Package service with digital twin event handling functions
package hubrouter

import (
	"fmt"
	"log/slog"
)

// HandleEventFlow agent sends an event
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
	dThingID, err := svc.dtwStore.UpdateEventValue(agentID, thingID, name, value, messageID)
	if err != nil {
		return err
	}
	if svc.tb == nil {
		return fmt.Errorf("HandleEventFlow: no transport binding")
	}
	// resubmit the event to subscribers of the digital twin in the background
	go svc.tb.PublishEvent(dThingID, name, value, messageID)
	return nil
}
