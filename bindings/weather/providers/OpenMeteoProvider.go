package providers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/weather/config"
	jsoniter "github.com/json-iterator/go"
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
			"&wind_speed_unit=ms&temperature_unit=celsius&precipitation_unit=mm"+
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
	c.Updated = utils.DecodeAsString(currentWeather["time"], 0)
	//updated,_ := dateparse.ParseAny(timeStamp)
	//c.Updated = utils.FormatNowUTCMilli()
	c.AtmoPressureMsl = utils.DecodeAsString(currentWeather["pressure_msl"], 0)
	c.AtmoPressureSurface = utils.DecodeAsString(currentWeather["surface_pressure"], 0)
	c.CloudCover = utils.DecodeAsString(currentWeather["cloud_cover"], 0)
	c.Humidity = utils.DecodeAsString(currentWeather["relative_humidity_2m"], 0)
	c.Precipitation = utils.DecodeAsString(currentWeather["precipitation"], 0)
	c.Rain = utils.DecodeAsString(currentWeather["rain"], 0)
	//c.Showers = utils.DecodeAsString(currentWeather["showers"], 0)
	c.Snowfall = utils.DecodeAsString(currentWeather["snowfall"], 0)
	c.Temperature = utils.DecodeAsString(currentWeather["temperature_2m"], 0)
	c.WindSpeed = utils.DecodeAsString(currentWeather["wind_speed_10m"], 0)
	c.WindHeading = utils.DecodeAsString(currentWeather["wind_direction_10m"], 0)
	c.WindGusts = utils.DecodeAsString(currentWeather["wind_gusts_10m"], 0)
	c.WeatherCode = utils.DecodeAsInt(currentWeather["weather_code"])

	return c, err
}

func (svc *OpenMeteoProvider) ReadHourlyForecast(loc config.WeatherLocation) (f HourlyWeatherForecast, err error) {
	return f, errors.New("not yet implemented")
}

func NewOpenMeteoProvider() *OpenMeteoProvider {
	provider := OpenMeteoProvider{
		baseURL: OpenMeteoBaseURL,
	}
	return &provider
}
