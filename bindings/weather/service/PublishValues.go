package service

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/hiveot/gocore/messaging"
	"github.com/hiveot/gocore/utils"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
)

// Poll for current and forecast weather updates and publish events.
// The recommended interval to call poll at is 1 minute.
//
// This uses the timestamp of the last poll for each location adds the configured interval
// and compares it with the current time.
func (svc *WeatherBinding) Poll() error {
	var err error
	now := time.Now()

	// poll for the 'current' weather at the locations
	svc.locationStore.ForEach(func(loc config.WeatherLocation) {
		var currentWeather providers.CurrentWeather
		var currentInterval int
		// each location can have its own interval
		svc.mux.RLock()
		lastPollTime, found := svc.lastCurrentPoll[loc.ID]
		svc.mux.RUnlock()
		currentInterval = loc.CurrentInterval
		if currentInterval <= svc.cfg.MinCurrentInterval {
			currentInterval = svc.cfg.DefaultCurrentInterval
		}
		nextPoll := lastPollTime.Add(time.Second * time.Duration(currentInterval))
		if !found || nextPoll.Before(now) {
			var err2 error
			provider, providerID := svc.GetProvider(loc.WeatherProvider)
			if provider == nil {
				// skip
				err2 = fmt.Errorf("weather provider '%s' for location '%s' not found",
					loc.WeatherProvider, loc.Name)
			} else {
				currentWeather, err2 = provider.ReadCurrent(loc)
			}

			if err2 == nil {
				svc.mux.Lock()
				svc.currentWeather[loc.ID] = currentWeather
				svc.mux.Unlock()
				slog.Info("Poll result",
					slog.String("location", loc.ID),
					slog.String("provider", providerID),
					slog.String("temp", currentWeather.Temperature),
					slog.String("showers", currentWeather.Showers),
				)
				err2 = PublishCurrent(svc.ag, loc.ID, currentWeather)
			}
			// keep track of the last error but don't stop polling
			if err2 != nil {
				err = err2
			}
			svc.mux.Lock()
			svc.lastCurrentPoll[loc.ID] = now
			svc.mux.Unlock()
		}
	})
	// poll for the 'hourly forecast' weather at the locations
	svc.locationStore.ForEach(func(loc config.WeatherLocation) {
		// each location can have its own interval
		svc.mux.RLock()
		lastPollTime, found := svc.lastHourlyForecastPoll[loc.ID]
		svc.mux.RUnlock()
		hourlyInterval := loc.HourlyInterval
		if hourlyInterval <= svc.cfg.MinForecastInterval {
			hourlyInterval = svc.cfg.DefaultHourlyForecastInterval
		}
		nextPoll := lastPollTime.Add(time.Second * time.Duration(hourlyInterval))
		if !found || nextPoll.Before(now) {
			//weatherForecast, err2 := svc.defaultProvider.ReadForecast(loc)
			//if err2 != nil {
			//	err = err2
			//} else {
			//	svc.forecasts[loc.ID] = weatherForecast
			//	slog.Info("Forecast result",
			//		slog.String("location", loc.ID),
			//		//slog.String("temp", weatherForecast.Temperature),
			//		//slog.String("showers", weatherForecast.Showers),
			//	)
			//}
			svc.mux.Lock()
			svc.lastHourlyForecastPoll[loc.ID] = now
			svc.mux.Unlock()
		}
	})
	return err
}

// PublishBindingProperties publish attributes and configuration of the binding
func PublishBindingProperties(ag *messaging.Agent, thingID string, cfg *config.WeatherConfig) error {
	propMap := map[string]any{
		PropNameWeatherProvider: cfg.DefaultProvider,
		PropNameUnitsWindSpeed:  cfg.WindSpeedUnits,
	}
	err := ag.PubProperties(thingID, propMap)
	return err
}

// PublishLocationProperties publish attributes and configuration of a location
func PublishLocationProperties(ag *messaging.Agent, thingID string, config config.WeatherLocation) error {
	propMap := map[string]any{
		PropNameCurrentEnabled:      config.CurrentEnabled,
		PropNameCurrentInterval:     config.CurrentInterval,
		PropNameHourlyEnabled:       config.HourlyEnabled,
		PropNameHourlyInterval:      config.HourlyInterval,
		PropNameWeatherProvider:     config.WeatherProvider,
		vocab.PropLocationLatitude:  config.Latitude,
		vocab.PropLocationLongitude: config.Longitude,
		vocab.PropLocationName:      config.Name,
	}
	err := ag.PubProperties(thingID, propMap)
	return err
}

// PublishCurrent publish events with the current weather
func PublishCurrent(ag *messaging.Agent, thingID string, current providers.CurrentWeather) error {
	err := ag.PubProperty(thingID, PropNameCurrentUpdated, current.Updated)
	if err != nil {
		return err
	}

	// convert wind speed to configured units
	windSpeed := utils.DecodeAsNumber(current.WindSpeed)

	// convert wind gusts to configured units
	windGusts := utils.DecodeAsNumber(current.WindGusts)

	_ = ag.PubEvent(thingID, vocab.PropEnvHumidity, current.Humidity)
	_ = ag.PubEvent(thingID, vocab.PropEnvPrecipitation, current.Precipitation)
	_ = ag.PubEvent(thingID, vocab.PropEnvPressureSeaLevel, current.AtmoPressureMsl)
	_ = ag.PubEvent(thingID, vocab.PropEnvPressureSurface, current.AtmoPressureSurface)
	_ = ag.PubEvent(thingID, vocab.PropEnvPrecipitationRain, current.Rain)
	_ = ag.PubEvent(thingID, vocab.PropEnvPrecipitationSnow, current.Snowfall)
	_ = ag.PubEvent(thingID, vocab.PropEnvPrecipitation, current.Precipitation)
	_ = ag.PubEvent(thingID, vocab.PropEnvTemperature, current.Temperature)
	_ = ag.PubEvent(thingID, vocab.PropEnvWindHeading, current.WindHeading)

	// todo: configure unit
	windGustsKph := math.Round(float64(windGusts) * 3.6)
	windSpeedKph := math.Round(float64(windSpeed) * 3.6)
	_ = ag.PubEvent(thingID, vocab.PropEnvWindGusts, windGustsKph) // m/s -> km/h
	_ = ag.PubEvent(thingID, vocab.PropEnvWindSpeed, windSpeedKph) // m/s -> km/h

	return err
}
