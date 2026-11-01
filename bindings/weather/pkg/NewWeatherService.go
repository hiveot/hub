package weather_pkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/bindings/weather"
	"github.com/hiveot/hub/bindings/weather/internal"
)

// NewWeatherService creates a new weaether service module
//
//	serviceID is the thingID of the weather service, "" for default
//	storePath where to store weather configuration
//	cfg weather configuration
func NewWeatherService(serviceID string, storePath string, cfg *weather.WeatherConfig) (api.IHiveCell, error) {
	svc, err := internal.NewWeatherService(serviceID, storePath, cfg)
	return svc, err
}
