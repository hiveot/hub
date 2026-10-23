package weatherpkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hub/bindings/weather"
	"github.com/hiveot/hub/bindings/weather/internal"
)

// NewWeatherService creates a new weaether service module
//
// co is the link to the consumer sending requests and consuming notifications, like an RC Agent or a server chain. nil to manually link.
// storePath where to store weather locations data
// cfg service configuration
func NewWeatherService(co api.IHiveCell, storePath string, cfg *weather.WeatherConfig) api.IHiveCell {
	svc := internal.NewWeatherService(co, storePath, cfg)
	return svc
}
