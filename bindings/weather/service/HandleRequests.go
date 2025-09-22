package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
)

// HandleRequest passes the action and config request to the associated Thing.
func (svc *WeatherBinding) handleRequest(req *messaging.RequestMessage,
	c messaging.IConnection) (resp *messaging.ResponseMessage) {

	if req.Operation == vocab.OpWriteProperty {
		return svc.handleConfigRequest(req, c)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	err := fmt.Errorf("handleActionRequest: unknown operation '%s' for thing '%s'", req.Operation, req.ThingID)
	resp = req.CreateResponse(nil, err)
	slog.Warn(resp.Error)
	return resp
}

func (svc *WeatherBinding) handleConfigRequest(req *messaging.RequestMessage,
	_ messaging.IConnection) (resp *messaging.ResponseMessage) {
	var err error
	slog.Info("handleConfigRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	config, found := svc.cfg.Locations[req.ThingID]
	if !found {
		resp = req.CreateResponse(nil, fmt.Errorf("handleConfigRequest: Location '%s' not found", req.ThingID))
		slog.Warn(resp.Error)
		return
	}

	switch req.Name {
	case PropNameCurrentEnabled:
		config.CurrentEnabled = tputils.DecodeAsBool(req.Input)
	case PropNameCurrentInterval:
		newInterval := tputils.DecodeAsInt(req.Input)
		if newInterval < svc.cfg.MinCurrentInterval {
			err = fmt.Errorf("invalid interval '%d seconds' for obtaining current weather", newInterval)
			break
		}
		config.CurrentInterval = newInterval
	case PropNameDefaultProvider:
		newProvider := tputils.DecodeAsString(req.Input, 0)
		_, isProvider := svc.cfg.Providers[newProvider]
		if !isProvider {
			err = fmt.Errorf("unknown weather provider: %s", newProvider)
			break
		}
		svc.cfg.DefaultProvider = newProvider
	case PropNameHourlyEnabled:
		config.HourlyEnabled = tputils.DecodeAsBool(req.Input)
	case vocab.PropLocationLatitude:
		config.Latitude = tputils.DecodeAsString(req.Input, 0)
	case vocab.PropLocationLongitude:
		config.Longitude = tputils.DecodeAsString(req.Input, 0)
	case vocab.PropLocationName:
		config.Name = tputils.DecodeAsString(req.Input, 0)
	default:
		err = fmt.Errorf("handleConfigRequest: '%s' is not a configuration", req.Name)
	}
	if err != nil {
		resp = req.CreateResponse(nil, err)
		slog.Warn(resp.Error)
	}
	return resp
}
