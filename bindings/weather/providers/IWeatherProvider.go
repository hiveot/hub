package providers

import "github.com/hiveot/hivehub/bindings/weather/config"

// definitions to be implemented by weather service providers

// CurrentWeather holds the response data for the current weather (non-forecast)
type CurrentWeather struct {
	Temperature         string `json:"temperature"`
	Humidity            string `json:"humidity"`
	AtmoPressureMsl     string `json:"pressureMsl"`
	AtmoPressureSurface string `json:"pressureSurface"`
	CloudCover          string `json:"cloudCover"`
	Rain                string `json:"rain"`          // mm/hour
	Snowfall            string `json:"snowfall"`      // mm/hour
	Precipitation       string `json:"precipitation"` // rain or snow
	Showers             string `json:"showers"`
	Updated             string `json:"updated"` // time the weather was updated
	WeatherCode         int    `json:"weatherCode"`
	WindHeading         string `json:"windHeading"` // 0-359 degrees
	WindGusts           string `json:"windGusts"`   // m/s
	WindSpeed           string `json:"windSpeed"`   // m/s
}

// HourlyWeatherForecast holds the hourly forecast
type HourlyWeatherForecast struct {
	Updated string `json:"updated"` // time the forecast was last updated
}

type IWeatherProvider interface {
	BaseURL() string
	ReadCurrent(loc config.WeatherLocation) (CurrentWeather, error)
	ReadHourlyForecast(loc config.WeatherLocation) (HourlyWeatherForecast, error)
}
