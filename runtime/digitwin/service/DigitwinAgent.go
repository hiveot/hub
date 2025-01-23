package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
	"log/slog"
)

type DigitwinAgent struct {
	svc           *DigitwinService
	dirHandler    transports.RequestHandler
	valuesHandler transports.RequestHandler
}

// HandleRequest handles digitwin services requests
// Including reading events, properties, actions from the digital twin.
func (agent *DigitwinAgent) HandleRequest(
	req *transports.RequestMessage, c transports.IConnection) (resp *transports.ResponseMessage) {

	if req.ThingID == digitwin.DirectoryDThingID {
		resp = agent.dirHandler(req, c)
	} else if req.ThingID == digitwin.ValuesDThingID {
		resp = agent.valuesHandler(req, c)
	} else {
		slog.Warn("HandleRequest: dThingID is not a service capability", "dThingID", req.ThingID)
		err := fmt.Errorf("%s is not a digitwin service capability", req.ThingID)
		resp = req.CreateResponse(nil, err)
	}
	return resp
}

// NewDigitwinAgent creates the agent that passes messages to the service api.
// This uses the tdd2go generated service action handlers.
func NewDigitwinAgent(svc *DigitwinService) *DigitwinAgent {
	agent := &DigitwinAgent{
		svc:           svc,
		dirHandler:    digitwin.NewHandleDirectoryRequest(svc.DirSvc),
		valuesHandler: digitwin.NewHandleValuesRequest(svc.ValuesSvc),
	}
	return agent
}
