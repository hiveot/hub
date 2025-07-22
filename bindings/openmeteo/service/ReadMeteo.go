package service

import (
	"fmt"
	"github.com/hiveot/hub/messaging/tputils"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// ReadCurrentWeather requests the current weather from open-meteo
func ReadCurrentWeather(baseURL string, config WeatherConfiguration) (*CurrentWeather, error) {
	var rawWeather []byte
	weather := CurrentWeather{}

	// Simulation file
	if strings.HasPrefix(baseURL, "file://") {
		filename := baseURL[7:]
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Unable to read meteo file", "err", err, "filename", filename)
			return nil, err
		}
		rawWeather = buffer
	} else {
		// http request
		timezone := "America/Los_Angeles"
		current :=
			"weather_code,rain,showers,snowfall," +
				"temperature_2m,relative_humidity_2m,pressure_msl,surface_pressure," +
				"wind_speed_10m,wind_direction_10m,wind_gusts_10m," +
				"cloud_cover" +
				"&minutely_15=lightning_potential"

		reqURL := fmt.Sprintf("%s?timezone=%s&latitude=%s&longitude=%s"+
			"&models=gem_seamless"+
			"&current=%s",
			baseURL, timezone, config.Latitude, config.Longitude, current)
		req, _ := http.NewRequest("GET", reqURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		rawWeather, err = io.ReadAll(resp.Body)
	}
	weatherJson := make(map[string]any)
	// temperature_2m
	err := jsoniter.Unmarshal(rawWeather, &weatherJson)
	if err == nil {
		currentWeather := weatherJson["current"].(map[string]interface{})
		weather.AtmoPressureMsl = tputils.DecodeAsString(currentWeather["pressure_msl"], 0)
		weather.AtmoPressureSurface = tputils.DecodeAsString(currentWeather["surface_pressure"], 0)
		weather.CloudCover = tputils.DecodeAsString(currentWeather["cloud_cover"], 0)
		weather.Humidity = tputils.DecodeAsString(currentWeather["relative_humidity_2m"], 0)
		weather.Rain = tputils.DecodeAsString(currentWeather["rain"], 0)
		weather.Showers = tputils.DecodeAsString(currentWeather["showers"], 0)
		weather.Snow = tputils.DecodeAsString(currentWeather["snowfall"], 0)
		weather.Temperature = tputils.DecodeAsString(currentWeather["temperature_2m"], 0)
		weather.WindSpeed = tputils.DecodeAsString(currentWeather["wind_speed_10m"], 0)
		weather.WindDirection = tputils.DecodeAsString(currentWeather["wind_direction_10m"], 0)
		weather.WindGusts = tputils.DecodeAsString(currentWeather["wind_gusts_10m"], 0)
		weather.WeatherCode = int(tputils.DecodeAsInt(currentWeather["weather_code"]))
	}
	return &weather, err
}
