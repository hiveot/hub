package providers

// WeatherLocationConfig holds the configuration for a weather location
type WeatherLocationConfig struct {
	ID           string
	LocationName string
	Latitude     string
	Longitude    string
	//
	// Enable/disable current weather lookup for this location
	CurrentEnabled bool
	// nr of seconds interval to poll current weather
	CurrentInterval int

	// Enable/disable current weather lookup for this location
	ForecastEnabled bool
	// nr of seconds interval to poll forecast
	ForecastInterval int
}
