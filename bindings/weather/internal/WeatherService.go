package internal

import (
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/weather"
	"github.com/hiveot/hub/bindings/weather/providers"
)

// the key under which custom Thing titles are stored in the state service
const weatherLocationsKey = "weatherLocations"
const DefaultWeatherProvider = providers.OpenMeteoProviderID
const DefaultWeatherServiceID = "weather"

// WeatherService is the hub protocol binding plugin for integrating with Open-Meteo weather provider.
type WeatherService struct {
	*thing.ExposedThing

	cfg *weather.WeatherConfig

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
func (svc *WeatherService) AddLocation(loc weather.WeatherLocation) error {
	err := svc.locationStore.Add(loc)
	if err == nil {
		svc.PublishLocationTD(loc)
	}
	return err
}

// GetProvider returns the provider with the given ID or the default provider when no ID is given
// If the ID is invalid then this returns nil
func (svc *WeatherService) GetProvider(providerID string) (providers.IWeatherProvider, string) {
	if providerID == "" {
		providerID = svc.cfg.DefaultProvider
	}
	provider := svc.providers[providerID]
	return provider, providerID
}

// heartbeat runs every second and publishes value updates at the right interval
func (svc *WeatherService) heartBeat() {
	err := svc.Poll()
	if err != nil {
		slog.Error("heartBeat error", "err", err.Error())
	}
}

// LocationStore access intended for testing
func (svc *WeatherService) LocationStore() *LocationStore {
	return svc.locationStore
}

// RemoveLocation removes a location
func (svc *WeatherService) RemoveLocation(thingID string) {
	svc.locationStore.Remove(thingID)
}

// Start the weather binding using the given agent
func (svc *WeatherService) Start() {

	svc.PublishBindingTD()
	svc.PublishLocationTDs()

	slog.Info("Starting heartBeat")
	svc.stopFn = utils.StartHeartbeat(time.Second*60, svc.heartBeat)

}

// Stop the service and heartbeat
func (svc *WeatherService) Stop() {
	if svc.stopFn != nil {
		svc.stopFn()
	}
	svc.locationStore.Close()
}

// NewWeatherService creates a new weather service binding
//
//	serviceID is the thingID of the weather service, "" for default
//	storePath where to store weather configuration
//	cfg weather configuration
func NewWeatherService(serviceID string, storePath string, cfg *weather.WeatherConfig) (*WeatherService, error) {

	if serviceID == "" {
		serviceID = DefaultWeatherServiceID
	}
	locationStore, err := NewLocationStore(storePath)
	if err != nil {
		return nil, err
	}
	// load the saved and add the pre-configured locations
	// Note that this will remove any modifications to the locations
	for id, loc := range cfg.Locations {
		loc.ID = id
		_ = locationStore.Add(loc)
	}

	// these are from hub configuration
	svc := &WeatherService{
		ExposedThing:           thing.NewExposedThing(serviceID, nil),
		cfg:                    cfg,
		providers:              make(map[string]providers.IWeatherProvider),
		locationStore:          locationStore,
		currentWeather:         make(map[string]providers.CurrentWeather),
		lastCurrentPoll:        make(map[string]time.Time),
		hourlyForecasts:        make(map[string]providers.HourlyWeatherForecast),
		lastHourlyForecastPoll: make(map[string]time.Time),
		stopFn:                 nil,
	}
	svc.providers[providers.OpenMeteoProviderID] = providers.NewOpenMeteoProvider()
	//svc.providers[providers.EnvCanadaProviderID] = providers.NewEnvCanadaProvider()

	return svc, err
}
