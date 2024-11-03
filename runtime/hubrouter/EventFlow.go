// Package hubrouter with digital twin event routing functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"log/slog"
)

// HandlePublishEvent agent sends an event
//
// This re-submits the event with the digital twin's thing ID in the background,
// and stores it in the digital twin instance. Only the last event value is kept.
func (svc *HubRouter) HandlePublishEvent(
	agentID string, thingID string, name string, value any, messageID string) error {
	slog.Debug("HandlePublishEventFlow (from agent)",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("eventName", name),
		slog.String("value", fmt.Sprintf("%v", value)),
		slog.String("messageID", messageID),
	)
	dThingID, err := svc.dtwStore.UpdateEventValue(agentID, thingID, name, value, messageID)
	if err != nil {
		return err
	}
	// resubmit the event to subscribers of the digital twin in the background
	go svc.cm.PublishEvent(dThingID, name, value, messageID, agentID)
	return nil
}

// HandleReadEvent consumer reads a digital twin thing's event value
func (svc *HubRouter) HandleReadEvent(consumerID string, thingID string, name string) (reply any, err error) {
	slog.Debug("HandleReadEvent", slog.String("consumerID", consumerID))
	reply, err = svc.dtwService.ValuesSvc.ReadEvent(consumerID,
		digitwin.ValuesReadEventArgs{ThingID: thingID, Name: name})
	return reply, err
}

// HandleReadAllEvents consumer reads all digital twin thing's event values
func (svc *HubRouter) HandleReadAllEvents(consumerID string, dThingID string) (reply any, err error) {
	slog.Debug("HandleReadEvent", slog.String("consumerID", consumerID))
	reply, err = svc.dtwService.ValuesSvc.ReadAllEvents(consumerID, dThingID)
	return reply, err
}
