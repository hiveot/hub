package service

import (
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/messaging"
	"log/slog"
	"time"
)

// the key under which custom Thing titles are stored in the state service
const weatherLocationsKey = "weatherLocations"
const DefaultWeatherProvider = providers.OpenMeteoProviderID

// WeatherBinding is the hub protocol binding plugin for integrating with Open-Meteo weather provider.
type WeatherBinding struct {
	cfg *config.WeatherConfig

	// hub client to publish TDs and values and receive actions
	ag *messaging.Agent

	// Supported weather providers
	//providers       map[string]IWeatherProvider
	defaultProvider providers.IWeatherProvider

	// The configured locationStore
	locationStore *LocationStore
	current       map[string]providers.CurrentWeather
	currentPoll   map[string]int // poll counter by location ID
	forecasts     map[string]providers.ForecastWeather
	forecastPoll  map[string]int // poll counter by location ID

	// stop the heartbeat
	stopFn func()
}

// LocationStore access intended for testing
func (svc *WeatherBinding) LocationStore() *LocationStore {
	return svc.locationStore
}

// Start the weather binding using the given agent
func (svc *WeatherBinding) Start(ag *messaging.Agent) error {
	svc.ag = ag

	// load the saved and add the pre-configured locations
	err := svc.locationStore.Open()
	for id, loc := range svc.cfg.Locations {
		loc.ID = id
		_ = svc.locationStore.Add(loc)
	}

	if err == nil {
		err = PublishBindingTD(ag)
	}
	if err == nil {
		err = PublishLocationTDs(ag, svc.cfg, svc.locationStore)
	}
	if err == nil {
		slog.Info("Starting heartBeat")
		svc.stopFn = plugin.StartHeartbeat(time.Second*900, svc.heartBeat)
	}
	if err != nil {
		svc.Stop()
	}

	return err
}

// heartbeat runs every second and publishes value updates at the right interval
func (svc *WeatherBinding) heartBeat() {
	err := svc.Poll()
	if err != nil {
		slog.Error("heartBeat error", "err", err.Error())
	}
}

// Stop the service and heartbeat
func (svc *WeatherBinding) Stop() {
	if svc.stopFn != nil {
		svc.stopFn()
	}
	svc.locationStore.Close()
}

// NewWeatherBinding creates a new Protocol Binding service
func NewWeatherBinding(storePath string, cfg *config.WeatherConfig) *WeatherBinding {

	// these are from hub configuration
	svc := &WeatherBinding{
		cfg:             cfg,
		defaultProvider: providers.NewOpenMeteoProvider(),
		ag:              nil,
		locationStore:   NewLocationStore(storePath),
		current:         make(map[string]providers.CurrentWeather),
		currentPoll:     make(map[string]int),
		forecasts:       make(map[string]providers.ForecastWeather),
		forecastPoll:    make(map[string]int),
		stopFn:          nil,
	}
	return svc
}
