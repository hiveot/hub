package service

import (
	"fmt"
	digitwin2 "github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
)

type DigitwinAgent struct {
	svc           *DigitwinService
	dirHandler    api.ActionHandler
	valuesHandler api.ActionHandler
}

// HandleAction digitwin services request (args order as generated by tdd2go)
func (agent *DigitwinAgent) HandleAction(
	consumerID string, dThingID string, actionName string, input any, messageID string) (
	status string, output any, err error) {

	if dThingID == digitwin2.DirectoryDThingID {
		status, output, err = agent.dirHandler(consumerID, dThingID, actionName, input, messageID)
	} else if dThingID == digitwin2.ValuesDThingID {
		status, output, err = agent.valuesHandler(consumerID, dThingID, actionName, input, messageID)
	} else {
		slog.Warn("HandleAction: dThingID is not a service capability", "dThingID", dThingID)
		err = fmt.Errorf("%s is not a digitwin service capability", dThingID)
		status = vocab.ProgressStatusFailed
	}
	return status, output, err
}

// NewDigitwinAgent creates the agent that passes messages to the service api.
// This uses the tdd2go generated service action handlers.
func NewDigitwinAgent(svc *DigitwinService) *DigitwinAgent {
	agent := &DigitwinAgent{
		svc:           svc,
		dirHandler:    digitwin2.NewHandleDirectoryAction(svc.DirSvc),
		valuesHandler: digitwin2.NewHandleValuesAction(svc.ValuesSvc),
	}
	return agent
}
