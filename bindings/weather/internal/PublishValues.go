package internal

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/weather"
	"github.com/hiveot/hub/bindings/weather/providers"
)

// Poll for current and forecast weather updates and publish events.
// The recommended interval to call poll at is 1 minute.
//
// This uses the timestamp of the last poll for each location adds the configured interval
// and compares it with the current time.
func (svc *WeatherService) Poll() error {
	var err error
	now := time.Now()

	// poll for the 'current' weather at the locations
	svc.locationStore.ForEach(func(loc weather.WeatherLocation) {
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
				svc.PublishCurrent(loc.ID, currentWeather)
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
	svc.locationStore.ForEach(func(loc weather.WeatherLocation) {
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
func (svc *WeatherService) PublishBindingProperties(thingID string) error {
	propMap := map[string]any{
		PropNameWeatherProvider: svc.cfg.DefaultProvider,
		PropNameUnitsWindSpeed:  svc.cfg.WindSpeedUnits,
	}
	svc.PubProperties(thingID, propMap, true)
	return nil
}

// PublishLocationProperties publish attributes and configuration of a location
func (svc *WeatherService) PublishLocationProperties(thingID string, config weather.WeatherLocation) error {
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
	svc.PubProperties(thingID, propMap, true)
	return nil
}

// PublishCurrent publish events with the current weather
func (svc *WeatherService) PublishCurrent(thingID string, current providers.CurrentWeather) {
	svc.PubProperty(thingID, PropNameCurrentUpdated, current.Updated, true)

	// convert wind speed to configured units
	windSpeed := utils.DecodeAsNumber(current.WindSpeed)

	// convert wind gusts to configured units
	windGusts := utils.DecodeAsNumber(current.WindGusts)

	svc.PubEvent(thingID, vocab.PropEnvHumidity, current.Humidity)
	svc.PubEvent(thingID, vocab.PropEnvPrecipitation, current.Precipitation)
	svc.PubEvent(thingID, vocab.PropEnvPressureSeaLevel, current.AtmoPressureMsl)
	svc.PubEvent(thingID, vocab.PropEnvPressureSurface, current.AtmoPressureSurface)
	svc.PubEvent(thingID, vocab.PropEnvPrecipitationRain, current.Rain)
	svc.PubEvent(thingID, vocab.PropEnvPrecipitationSnow, current.Snowfall)
	svc.PubEvent(thingID, vocab.PropEnvPrecipitation, current.Precipitation)
	svc.PubEvent(thingID, vocab.PropEnvTemperature, current.Temperature)
	svc.PubEvent(thingID, vocab.PropEnvWindHeading, current.WindHeading)

	// todo: configure unit
	windGustsKph := math.Round(float64(windGusts) * 3.6)
	windSpeedKph := math.Round(float64(windSpeed) * 3.6)
	svc.PubEvent(thingID, vocab.PropEnvWindGusts, windGustsKph) // m/s -> km/h
	svc.PubEvent(thingID, vocab.PropEnvWindSpeed, windSpeedKph) // m/s -> km/h
}
