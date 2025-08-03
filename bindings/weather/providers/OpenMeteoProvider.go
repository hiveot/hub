package providers

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging/tputils"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// TODO: support reading simulation from file for testing
const OpenMeteoProviderID = "open-meteo"
const OpenMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"

// OpenMeteoProvider implementing IWeatherProvider
type OpenMeteoProvider struct {
	baseURL string
}

func (svc *OpenMeteoProvider) BaseURL() string {
	return svc.baseURL
}

// ReadCurrent requests the current weather from Open-Meteo provider
func (svc *OpenMeteoProvider) ReadCurrent(config config.WeatherLocation) (c CurrentWeather, err error) {
	var rawWeather []byte

	// Simulation file
	if strings.HasPrefix(svc.baseURL, "file://") {
		filename := svc.baseURL[7:]
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Unable to read meteo file", "err", err, "filename", filename)
			return c, err
		}
		rawWeather = buffer
	} else {
		// http request
		timezone := "America/Los_Angeles"
		current :=
			"weather_code,precipitation,rain,showers,snowfall," +
				"temperature_2m,relative_humidity_2m,pressure_msl,surface_pressure," +
				"wind_speed_10m,wind_direction_10m,wind_gusts_10m," +
				"cloud_cover" +
				"&minutely_15=lightning_potential"

		reqURL := fmt.Sprintf("%s?timezone=%s&latitude=%s&longitude=%s"+
			"&models=gem_seamless"+
			"&current=%s",
			svc.baseURL, timezone, config.Latitude, config.Longitude, current)
		req, _ := http.NewRequest("GET", reqURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return c, err
		}
		rawWeather, err = io.ReadAll(resp.Body)
	}
	weatherJson := make(map[string]any)
	// temperature_2m
	err = jsoniter.Unmarshal(rawWeather, &weatherJson)
	if err == nil {
		_, found := weatherJson["error"]
		if found {
			reason, _ := weatherJson["reason"].(string)
			slog.Error("Open-Meteo error:",
				"err", reason)
			err = errors.New(reason)
		}
	}
	if err != nil {
		return c, err
	}
	c.Updated = utils.FormatNowUTCMilli()
	currentWeather := weatherJson["current"].(map[string]interface{})
	c.AtmoPressureMsl = tputils.DecodeAsString(currentWeather["pressure_msl"], 0)
	c.AtmoPressureSurface = tputils.DecodeAsString(currentWeather["surface_pressure"], 0)
	c.CloudCover = tputils.DecodeAsString(currentWeather["cloud_cover"], 0)
	c.Humidity = tputils.DecodeAsString(currentWeather["relative_humidity_2m"], 0)
	c.Precipitation = tputils.DecodeAsString(currentWeather["precipitation"], 0)
	c.Rain = tputils.DecodeAsString(currentWeather["rain"], 0)
	//c.Showers = tputils.DecodeAsString(currentWeather["showers"], 0)
	c.Snow = tputils.DecodeAsString(currentWeather["snowfall"], 0)
	c.Temperature = tputils.DecodeAsString(currentWeather["temperature_2m"], 0)
	c.WindSpeed = tputils.DecodeAsString(currentWeather["wind_speed_10m"], 0)
	c.WindDirection = tputils.DecodeAsString(currentWeather["wind_direction_10m"], 0)
	c.WindGusts = tputils.DecodeAsString(currentWeather["wind_gusts_10m"], 0)
	c.WeatherCode = int(tputils.DecodeAsInt(currentWeather["weather_code"]))

	return c, err
}

func (svc *OpenMeteoProvider) ReadForecast(loc config.WeatherLocation) (f ForecastWeather, err error) {
	return f, errors.New("not yet implemented")
}

func NewOpenMeteoProvider() *OpenMeteoProvider {
	provider := OpenMeteoProvider{
		baseURL: OpenMeteoBaseURL,
	}
	return &provider
}
