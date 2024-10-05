package service

import (
	"fmt"
	digitwin2 "github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin"
	"log/slog"
)

type DigitwinAgent struct {
	svc        *DigitwinService
	dirHandler api.ActionHandler
}

// HandleAction authn services action request
func (agent *DigitwinAgent) HandleAction(
	consumerID string, dThingID string, actionName string, input any, messageID string) (
	status string, output any, err error) {

	if dThingID == digitwin2.DirectoryDThingID {
		status, output, err = agent.dirHandler(consumerID, dThingID, actionName, input, messageID)
		//} else if dThingID == digitwin2.ServiceDThingID {
		//	status, output, err = agent.svcHandler(consumerID, dThingID, actionName, input, messageID)
	} else {
		slog.Warn("HandleAction: dThingID is not a service capability", "dThingID", dThingID)
		err = fmt.Errorf("%s is not a digitwin service capability", dThingID)
	}
	if err != nil {
		status = digitwin.StatusFailed
	}
	return status, output, err
}

func NewDigitwinAgent(svc *DigitwinService) *DigitwinAgent {
	agent := &DigitwinAgent{
		svc:        svc,
		dirHandler: digitwin2.NewHandleDirectoryAction(svc.DirSvc),
	}
	return agent
}
