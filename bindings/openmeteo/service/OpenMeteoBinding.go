package service

import (
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/messaging"
	"log/slog"
	"time"
)

// the key under which custom Thing titles are stored in the state service
const weatherLocationsKey = "weatherLocations"

// TODO: support reading simulation from file for testing
const baseURL = "https://api.open-meteo.com/v1/forecast"

// OpenMeteoBinding is the hub protocol binding plugin for integrating with Open-Meteo weather provider.
type OpenMeteoBinding struct {

	// hub client to publish TDs and values and receive actions
	ag *messaging.Agent

	// The configured locations
	locations *LocationStore
	current   map[string]CurrentWeather
	forecasts map[string]ForecastWeather

	// stop the heartbeat
	stopFn func()
}

// Locations access intended for testing
func (svc *OpenMeteoBinding) Locations() *LocationStore {
	return svc.locations
}

// Poll for weather updates
func (svc *OpenMeteoBinding) Poll() error {
	var err error

	svc.locations.ForEach(func(loc WeatherConfiguration) {
		currentWeather, err2 := ReadCurrentWeather(baseURL, loc)
		if err2 != nil {
			err = err2
		} else {
			slog.Info("read weather", "location ID", loc.ID,
				"temp", currentWeather.Temperature)
		}
	})
	return err
}

func (svc *OpenMeteoBinding) Start(ag *messaging.Agent) error {
	svc.ag = ag

	err := svc.locations.Open()
	if err != nil {
		slog.Info("Starting heartBeat")
		svc.stopFn = plugin.StartHeartbeat(time.Second*900, svc.heartBeat)
	}
	return err
}

// heartbeat polls the EDS server every X seconds and publishes TD and value updates
func (svc *OpenMeteoBinding) heartBeat() {
	err := svc.Poll()
	if err != nil {
		slog.Error("heartBeat error", "err", err.Error())
	}
}

// Stop the service and heartbeat
func (svc *OpenMeteoBinding) Stop() {
	if svc.stopFn != nil {
		svc.stopFn()
	}
	svc.locations.Close()
}

// NewOpenMeteoBinding creates a new Protocol Binding service
func NewOpenMeteoBinding(storePath string) *OpenMeteoBinding {

	// these are from hub configuration
	svc := &OpenMeteoBinding{
		ag:        nil,
		locations: NewLocationStore(storePath),
		current:   make(map[string]CurrentWeather),
		forecasts: make(map[string]ForecastWeather),
		stopFn:    nil,
	}
	return svc
}
