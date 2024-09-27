// Package service with digital twin property handling functions
package service

import (
	"fmt"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// UpdatePropertyValue updates the last known thing property value.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// Invoked by agents
func (svc *DigitwinService) UpdatePropertyValue(
	agentID string, thingID string, propName string, value any) error {

	slog.Info("UpdatePropertyValue",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("propName", propName),
		slog.String("value", fmt.Sprintf("%v", value)),
	)
	err := svc.dtwStore.UpdatePropertyValue(agentID, thingID, propName, value)
	return err
}

// WriteProperty handles the request to write a new value to a property
func (svc *DigitwinService) WriteProperty(
	consumerID string, dThingID string, name string, newValue any) (status string, err error) {

	slog.Info("UpdatePropertyValue",
		slog.String("consumerID", consumerID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("newValue", fmt.Sprintf("%v", newValue)),
	)
	// update the store
	status = digitwin.StatusPending
	err = svc.dtwStore.WriteProperty(consumerID, dThingID, name, newValue, status)

	// forward the request to the thing's agent
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	status, found, err := svc.tb.WriteProperty(agentID, thingID, name, newValue)
	if err != nil || !found {
		status = digitwin.StatusFailed
	}
	// update the status
	_ = svc.dtwStore.WriteProperty(consumerID, dThingID, name, newValue, status)

	return status, err
}
