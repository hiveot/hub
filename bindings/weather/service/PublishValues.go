package service

import (
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"log/slog"
)

// Poll for current and forecast weather updates and publish events
// This iterates each location and polls at the configured interval
func (svc *WeatherBinding) Poll() error {
	var err error

	// poll for the 'current' weather at the locations
	svc.locationStore.ForEach(func(loc config.WeatherLocation) {
		// each location can have its own interval
		currentPollCounter, found := svc.currentPoll[loc.ID]
		currentInterval := loc.CurrentInterval
		if currentInterval <= svc.cfg.MinCurrentInterval {
			currentInterval = svc.cfg.DefaultCurrentInterval
		}
		if !found || currentPollCounter >= currentInterval {
			currentWeather, err2 := svc.defaultProvider.ReadCurrent(loc)
			if err2 == nil {
				svc.current[loc.ID] = currentWeather
				slog.Info("Poll result",
					slog.String("location", loc.ID),
					slog.String("temp", currentWeather.Temperature),
					slog.String("showers", currentWeather.Showers),
				)
				err2 = svc.PublishCurrent(loc.ID, currentWeather)
			}
			if err2 != nil {
				err = err2
			}
			currentPollCounter = 0
		}
		currentPollCounter++
		svc.currentPoll[loc.ID] = currentPollCounter
	})
	// poll for the 'forecast' weather at the locations
	svc.locationStore.ForEach(func(loc config.WeatherLocation) {
		// each location can have its own interval
		pollCounter, found := svc.forecastPoll[loc.ID]
		forecastInterval := loc.ForecastInterval
		if forecastInterval <= svc.cfg.MinForecastInterval {
			forecastInterval = svc.cfg.DefaultForecastInterval
		}
		if !found || pollCounter >= forecastInterval {
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
			pollCounter = 0
		}
		pollCounter++
		svc.forecastPoll[loc.ID] = pollCounter
	})
	return err
}

// PublishCurrent publish events with the current weather
func (svc *WeatherBinding) PublishCurrent(thingID string, current providers.CurrentWeather) error {

	err := svc.ag.PubEvent(thingID, "humidity", current.Humidity)
	err = svc.ag.PubEvent(thingID, "precipitation", current.Precipitation)
	err = svc.ag.PubEvent(thingID, "pressureMsl", current.AtmoPressureMsl)
	err = svc.ag.PubEvent(thingID, "pressureSurface", current.AtmoPressureSurface)
	err = svc.ag.PubEvent(thingID, "rain", current.Rain)
	err = svc.ag.PubEvent(thingID, "temperature", current.Temperature)
	err = svc.ag.PubEvent(thingID, "windDirection", current.WindDirection)
	err = svc.ag.PubEvent(thingID, "windSpeed", current.WindSpeed)

	return err
}
