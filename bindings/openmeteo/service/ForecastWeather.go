package service

// CurrentWeather holds the open-meteo weather response object
type CurrentWeather struct {
	Temperature         string
	Humidity            string
	AtmoPressureMsl     string
	AtmoPressureSurface string
	CloudCover          string
	Rain                string
	Snow                string
	Showers             string
	WeatherCode         int
	WindDirection       string // 0-359 degrees
	WindGusts           string // m/sec
	WindSpeed           string // m/sec
}

type ForecastWeather struct {
}
