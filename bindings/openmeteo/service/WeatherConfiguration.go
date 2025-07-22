package service

// WeatherConfiguration holds the configuration for a weather location
type WeatherConfiguration struct {
	ID           string
	LocationName string
	Latitude     string
	Longitude    string
	//
	// Enable/disable current weather lookup for this location
	CurrentWeather bool
	// Enable/disable current weather lookup for this location
	DailyForecast bool
}
