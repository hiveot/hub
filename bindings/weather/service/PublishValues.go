package service

import (
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"log/slog"
	"time"
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
		// each location can have its own interval
		svc.mux.RLock()
		lastPollTime, found := svc.lastCurrentPoll[loc.ID]
		svc.mux.RUnlock()
		currentInterval := loc.CurrentInterval
		if currentInterval <= svc.cfg.MinCurrentInterval {
			currentInterval = svc.cfg.DefaultCurrentInterval
		}
		nextPoll := lastPollTime.Add(time.Second * time.Duration(currentInterval))
		if !found || nextPoll.Before(now) {
			currentWeather, err2 := svc.defaultProvider.ReadCurrent(loc)
			if err2 == nil {
				svc.mux.Lock()
				svc.current[loc.ID] = currentWeather
				svc.mux.Unlock()
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
			svc.mux.Lock()
			svc.lastCurrentPoll[loc.ID] = now
			svc.mux.Unlock()
		}
	})
	// poll for the 'forecast' weather at the locations
	svc.locationStore.ForEach(func(loc config.WeatherLocation) {
		// each location can have its own interval
		svc.mux.RLock()
		lastPollTime, found := svc.lastForecastPoll[loc.ID]
		svc.mux.RUnlock()
		forecastInterval := loc.ForecastInterval
		if forecastInterval <= svc.cfg.MinForecastInterval {
			forecastInterval = svc.cfg.DefaultForecastInterval
		}
		nextPoll := lastPollTime.Add(time.Second * time.Duration(forecastInterval))
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
			svc.lastForecastPoll[loc.ID] = now
			svc.mux.Unlock()
		}
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
