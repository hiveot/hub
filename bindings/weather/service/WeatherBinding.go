package service

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/gocore/messaging"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/lib/plugin"
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
	providers map[string]providers.IWeatherProvider

	// The configured locationStore
	locationStore          *LocationStore
	currentWeather         map[string]providers.CurrentWeather
	lastCurrentPoll        map[string]time.Time // poll timestamp by location ID
	hourlyForecasts        map[string]providers.HourlyWeatherForecast
	lastHourlyForecastPoll map[string]time.Time // poll timestamp by location ID

	// protect r/w of current weather
	mux sync.RWMutex

	// stop the heartbeat
	stopFn func()
}

// AddLocation adds a new location and publishes its TD
func (svc *WeatherBinding) AddLocation(loc config.WeatherLocation) error {
	err := svc.locationStore.Add(loc)
	if err == nil {
		err = PublishLocationTD(svc.ag, svc.cfg, loc)
	}
	return err
}

// GetProvider returns the provider with the given ID or the default provider when no ID is given
// If the ID is invalid then this returns nil
func (svc *WeatherBinding) GetProvider(providerID string) (providers.IWeatherProvider, string) {
	if providerID == "" {
		providerID = svc.cfg.DefaultProvider
	}
	provider := svc.providers[providerID]
	return provider, providerID
}

// heartbeat runs every second and publishes value updates at the right interval
func (svc *WeatherBinding) heartBeat() {
	err := svc.Poll()
	if err != nil {
		slog.Error("heartBeat error", "err", err.Error())
	}
}

// LocationStore access intended for testing
func (svc *WeatherBinding) LocationStore() *LocationStore {
	return svc.locationStore
}

// RemoveLocation removes a location
func (svc *WeatherBinding) RemoveLocation(thingID string) {
	svc.locationStore.Remove(thingID)
}

// Start the weather binding using the given agent
func (svc *WeatherBinding) Start(ag *messaging.Agent) error {
	svc.ag = ag

	// load the saved and add the pre-configured locations
	// Note that this will remove any modifications
	err := svc.locationStore.Open()
	for id, loc := range svc.cfg.Locations {
		loc.ID = id
		_ = svc.locationStore.Add(loc)
	}

	if err == nil {
		err = PublishBindingTD(ag, svc.cfg)
	}
	if err == nil {
		err = PublishLocationTDs(ag, svc.cfg, svc.locationStore)
	}
	if err == nil {
		slog.Info("Starting heartBeat")
		svc.stopFn = plugin.StartHeartbeat(time.Second*60, svc.heartBeat)

		// handle config requests
		ag.SetRequestHandler(svc.handleRequest)
	}
	if err != nil {
		svc.Stop()
	}

	return err
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
		cfg:                    cfg,
		providers:              make(map[string]providers.IWeatherProvider),
		ag:                     nil,
		locationStore:          NewLocationStore(storePath),
		currentWeather:         make(map[string]providers.CurrentWeather),
		lastCurrentPoll:        make(map[string]time.Time),
		hourlyForecasts:        make(map[string]providers.HourlyWeatherForecast),
		lastHourlyForecastPoll: make(map[string]time.Time),
		stopFn:                 nil,
	}
	svc.providers[providers.OpenMeteoProviderID] = providers.NewOpenMeteoProvider()
	//svc.providers[providers.EnvCanadaProviderID] = providers.NewEnvCanadaProvider()
	return svc
}
