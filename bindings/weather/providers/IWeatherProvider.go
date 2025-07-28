package providers

import "github.com/hiveot/hub/bindings/weather/config"

// definitions to be implemented by weather service providers

// CurrentWeather holds the weather provider response data
type CurrentWeather struct {
	Temperature         string `json:"temperature"`
	Humidity            string `json:"humidity"`
	AtmoPressureMsl     string `json:"pressureMsl"`
	AtmoPressureSurface string `json:"pressureSurface"`
	CloudCover          string `json:"cloudCover"`
	Rain                string `json:"rain"`
	Snow                string `json:"snow"`
	Precipitation       string `json:"precipitation"` // rain or snow
	Showers             string `json:"showers"`
	Updated             string `json:"updated"` // time the weather was updated
	WeatherCode         int    `json:"weatherCode"`
	WindDirection       string `json:"windDirection"` // 0-359 degrees
	WindGusts           string `json:"windGusts"`     // m/sec
	WindSpeed           string `json:"windSpeed"`     // m/sec
}

// ForecastWeather holds the provided forecast
type ForecastWeather struct {
	Updated string `json:"updated"` // time the forecast was updated
}

type IWeatherProvider interface {
	BaseURL() string
	ReadCurrent(loc config.WeatherLocation) (CurrentWeather, error)
	ReadForecast(loc config.WeatherLocation) (ForecastWeather, error)
}
