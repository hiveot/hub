package config

// WeatherConfig holds the configuration of the weather binding service
type WeatherConfig struct {

	// The default weather provider if not specified in the location
	DefaultProvider string `yaml:"defaultProvider"`

	// Default polling interval for current weather in seconds
	DefaultCurrentInterval        int `yaml:"defaultCurrentInterval,omitempty"`
	DefaultHourlyForecastInterval int `yaml:"defaultForecastInterval,omitempty"`

	// Minimum allowable polling interval for current weather in seconds
	MinCurrentInterval  int `yaml:"minCurrentInterval,omitempty"`
	MinForecastInterval int `yaml:"minForecastInterval,omitempty"`

	Providers map[string]WeatherProvider `yaml:"providers"`

	// locations by thingID
	Locations map[string]WeatherLocation `yaml:"locations"`
}

// NewWeatherConfig creates a new default weather binding configuration
func NewWeatherConfig() *WeatherConfig {
	cfg := WeatherConfig{
		DefaultProvider:               "open-meteo",
		DefaultCurrentInterval:        15 * 60,
		DefaultHourlyForecastInterval: 60 * 60,
		MinCurrentInterval:            5 * 60,
		MinForecastInterval:           10 * 60,
	}
	return &cfg
}

// WeatherLocation holds the configuration for a weather location
// A location can be added through the binding config.
// (future plan is to allow managing locations using actions)
type WeatherLocation struct {
	ID          string `yaml:"ID,omitempty"`
	Description string `yaml:"description,omitempty"`
	Latitude    string `yaml:"latitude"`
	Longitude   string `yaml:"longitude"`
	Name        string `yaml:"name"`

	// Enable/disable current weather lookup for this location
	CurrentEnabled bool `yaml:"currentEnabled,omitempty"`
	// override nr of seconds interval to poll current weather
	CurrentInterval int `yaml:"currentInterval,omitempty"`

	// Enable/disable current weather lookup for this location
	HourlyEnabled bool `yaml:"forecastEnabled,omitempty"`
	// HourlyInterval interval to obtain the next hourly forecast. Default 3600
	HourlyInterval int `yaml:"hourlyInterval,omitempty"`

	// Provider optionally overrides the default provider
	Provider string `yaml:"provider,omitempty"`
}

// WeatherProvider defined access to a weather service
type WeatherProvider struct {
	// Name of the weather provider
	Name string `yaml:"name"`
	// The base URL for this weather provider. Defaults to the baked-in URL
	BaseURL string `yaml:"baseURL,omitempty"`
	// API-Key for licensed users
	ApiKey string `yaml:"apiKey,omitempty"`
}
