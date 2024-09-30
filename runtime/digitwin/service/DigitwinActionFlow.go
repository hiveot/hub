// Package service with digital twin action flow handling functions
package service

import (
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// InvokeAction stores and passes an action request to the thing agent
// This passes it on to the actual device or service.
func (svc *DigitwinService) InvokeAction(
	consumerID string, dThingID string, actionName string, input any, messageID string) (output any, status string, err error) {
	slog.Info("InvokeAction")

	err = svc.dtwStore.UpdateActionStart(consumerID, dThingID, actionName, input, "pending")
	if err != nil {
		slog.Warn("InvokeAction failed.", "err", err.Error())
		return nil, digitwin.StatusFailed, err
	}
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	// pass the action to the agent and return the output
	if svc.tb != nil {
		// FIXME: make WoT interoperable
		status, output, err = svc.tb.InvokeAction(agentID, thingID, actionName, input, messageID)
	}
	return output, status, err
}

// UpdateActionProgress updates the progress of an action invocation
//
//	agentID sending the update
//	thingID handling the action
//	actionName to invoke
//	status of the action
//	output in case status is completed
func (svc *DigitwinService) UpdateActionProgress(
	agentID string, thingID string, actionName string, status string, output any) error {

	err := svc.dtwStore.UpdateActionProgress(agentID, thingID, actionName, status, output)
	return err
}
