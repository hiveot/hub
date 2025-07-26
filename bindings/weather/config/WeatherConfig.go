package config

// WeatherConfig holds the configuration of the weather service
type WeatherConfig struct {
	// The default weather provider if not specified in the location
	DefaultProvider string `yaml:"defaultProvider"`

	// Default polling interval for current weather in seconds
	DefaultCurrentInterval  int
	DefaultForecastInterval int

	// Minimum allowable polling interval for current weather in seconds
	MinCurrentInterval  int
	MinForecastInterval int
}

// NewWeatherConfig creates a new default weather binding configuration
func NewWeatherConfig() *WeatherConfig {
	cfg := WeatherConfig{
		DefaultProvider:         "", // set in app
		DefaultCurrentInterval:  15 * 60,
		DefaultForecastInterval: 60 * 60,
		MinCurrentInterval:      5 * 60,
		MinForecastInterval:     10 * 60,
	}
	return &cfg
}
