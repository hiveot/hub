// Package service with digital twin property handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// HandleUpdatePropertyFlow updates the last known thing property value and
// publishes property changes to observers.
//
// agentID is the ID of the agent sending the update
// thingID is the ID of the original thing as managed by the agent.
// Invoked by agents
func (svc *HubRouter) HandleUpdatePropertyFlow(
	agentID string, thingID string, propName string, value any, messageID string) error {

	slog.Info("UpdatePropertyValue",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID),
		slog.String("propName", propName),
		slog.String("value", fmt.Sprintf("%v", value)),
	)
	err := svc.dtwStore.UpdatePropertyValue(agentID, thingID, propName, value, messageID)
	return err
}

// HandleWritePropertyFlow handles the request to write a new value to a property
func (svc *HubRouter) HandleWritePropertyFlow(
	consumerID string, dThingID string, name string, newValue any) (
	status string, messageID string, err error) {

	slog.Info("UpdatePropertyValue",
		slog.String("consumerID", consumerID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("newValue", fmt.Sprintf("%v", newValue)),
	)
	// update the store
	status = digitwin.StatusPending
	err = svc.dtwStore.WriteProperty(consumerID, dThingID, name, newValue, status, messageID)

	// forward the request to the thing's agent
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	if svc.tb != nil {
		status, err = svc.tb.WriteProperty(agentID, thingID, name, newValue, messageID)
		// save the new status
		_ = svc.dtwStore.WriteProperty(consumerID, dThingID, name, newValue, status, messageID)
	} else {
		status = digitwin.StatusFailed
	}
	return status, messageID, err
}
