package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivehub/api/go/vocab"
	"github.com/hiveot/hivehub/bindings/weather/config"
	"github.com/hiveot/hivekitgo/messaging"
	"github.com/hiveot/hivekitgo/utils"
)

// HandleRequest passes the action and config request to the associated Thing.
func (svc *WeatherBinding) handleRequest(req *messaging.RequestMessage,
	c messaging.IConnection) (resp *messaging.ResponseMessage) {
	var err error

	if req.Operation == vocab.OpWriteProperty {
		return svc.handleConfigRequest(req, c)
	}

	slog.Info("handleActionRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	if req.Name == ActionNameAddLocation {
		err = fmt.Errorf("add location '%s' is not yet supported", req.ThingID)
		locInfo := config.WeatherLocation{}
		err = utils.DecodeAsObject(req.Input, &locInfo)
		if err == nil {
			err = svc.AddLocation(locInfo)
		}
	} else if req.Name == ActionNameRemoveLocation {
		err = fmt.Errorf("remove location '%s' is not yet supported", req.ThingID)
		svc.RemoveLocation(req.ThingID)
	} else {
		err = fmt.Errorf("handleActionRequest: unknown operation '%s' for thing '%s'", req.Operation, req.ThingID)
	}
	resp = req.CreateResponse(nil, err)
	slog.Warn(resp.Error.String())
	return resp
}

func (svc *WeatherBinding) handleConfigRequest(req *messaging.RequestMessage,
	_ messaging.IConnection) (resp *messaging.ResponseMessage) {
	var err error
	var newValue any // if newValue is set the property is published

	slog.Info("handleConfigRequest",
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID))

	// If this is a preconfigured location it cannot be modified
	loc, found := svc.locationStore.Get(req.ThingID)
	if !found {
		resp = req.CreateResponse(nil, fmt.Errorf("handleConfigRequest: Location '%s' not found", req.ThingID))
		slog.Warn(resp.Error.String())
		return resp
	}

	switch req.Name {
	case PropNameCurrentEnabled:
		loc.CurrentEnabled = utils.DecodeAsBool(req.Input)
		newValue = loc.CurrentEnabled
	case PropNameCurrentInterval:
		newInterval := utils.DecodeAsInt(req.Input)
		if newInterval < 0 {
			newInterval = svc.cfg.DefaultCurrentInterval
		} else if newInterval < svc.cfg.MinCurrentInterval {
			err = fmt.Errorf("invalid interval '%d seconds' for obtaining current weather", newInterval)
			break
		}
		loc.CurrentInterval = newInterval
		newValue = newInterval
	case PropNameWeatherProvider:
		newProvider := utils.DecodeAsString(req.Input, 0)
		_, isProvider := svc.cfg.Providers[newProvider]
		if !isProvider {
			err = fmt.Errorf("unknown weather provider: %s", newProvider)
			break
		}
		loc.WeatherProvider = newProvider
		newValue = newProvider
	case PropNameHourlyEnabled:
		loc.HourlyEnabled = utils.DecodeAsBool(req.Input)
		newValue = loc.HourlyEnabled
	case vocab.PropLocationLatitude:
		loc.Latitude = utils.DecodeAsString(req.Input, 0)
		newValue = loc.Latitude
	case vocab.PropLocationLongitude:
		loc.Longitude = utils.DecodeAsString(req.Input, 0)
		newValue = loc.Longitude
	case vocab.PropLocationName:
		loc.Name = utils.DecodeAsString(req.Input, 0)
		newValue = loc.Name
	default:
		err = fmt.Errorf("handleConfigRequest: '%s' is not a configuration", req.Name)
	}
	if err != nil {
		slog.Warn(err.Error())
	}
	resp = req.CreateResponse(req.Input, err)

	// If a new value is set, update the location and publish the result
	if err == nil && newValue != nil {
		svc.locationStore.Update(loc)
		go func() {
			_ = svc.ag.PubProperty(req.ThingID, req.Name, newValue)
		}()
	}
	return resp
}
