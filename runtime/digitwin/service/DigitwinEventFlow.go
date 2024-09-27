// Package service with digital twin event handling functions
package service

import (
	"fmt"
	"log/slog"
)

// AddEventValue adds a new event value to a digitwin instance
// Only the last event is kept.
func (svc *DigitwinService) AddEventValue(
	agentID string, thingID string, eventName string, value any) error {
	slog.Info("AddEvent",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("eventName", eventName),
		slog.String("value", fmt.Sprintf("%v", value)),
	)
	err := svc.dtwStore.UpdateEventValue(agentID, thingID, eventName, value)
	return err
}
