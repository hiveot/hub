package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
)

const EventNameCurrentWeather = "current"
const EventNameForecastWeather = "forecast"

// CreateBindingTD creates a Thing TD of this service
// This binding exposes the TD of itself.
func CreateBindingTD(serviceID string) *td.TD {
	// the agent-ID is the thingID of the binding
	tdoc := td.NewTD(serviceID, "Weather binding", vocab.ThingService)
	tdoc.Description = "Binding for the weather service"

	// The defaults are defined in the WeatherConfig object.
	prop := tdoc.AddProperty("defaultProvider", "Default Provider",
		"Name of the current default weather provider used",
		vocab.WoTDataTypeString)

	//prop = tdoc.AddProperty("publishChanges", "Only Publish Changes",
	//	"Only publish changes of the weather since previous poll",
	//	vocab.WoTDataTypeAnyURI)
	_ = prop
	return tdoc
}

// CreateTDOfLocation creates a Thing TD describing a weather location
func CreateTDOfLocation(defaultCfg *config.WeatherConfig, cfg *config.WeatherLocation) *td.TD {
	thingID := cfg.ID
	deviceType := vocab.ThingSensorEnvironment
	title := cfg.Name
	tdoc := td.NewTD(thingID, title, deviceType)

	prop := tdoc.AddProperty("currentEnabled", "Enable Current Weather",
		"Enable read the current weather",
		vocab.WoTDataTypeBool)
	prop.ReadOnly = false

	prop = tdoc.AddProperty("currentInterval", "Current Weather Updates",
		"Interval in seconds the current weather is updated",
		vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond
	prop.Minimum = float64(defaultCfg.MinCurrentInterval)
	prop.Default = defaultCfg.DefaultCurrentInterval
	prop.ReadOnly = false

	prop = tdoc.AddProperty("currentUpdated", "Current Weather Updated",
		"Time the current weather was last updated",
		vocab.WoTDataTypeDateTime)
	prop.ReadOnly = true

	prop = tdoc.AddProperty("forecastEnabled", "Enable Weather Forecast",
		"Enable reading the weather forecast",
		vocab.WoTDataTypeBool)
	prop.ReadOnly = false

	prop = tdoc.AddProperty("forecastInterval", "Weather Forecast Updates",
		"Interval in seconds the weather forecast is updated",
		vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond
	prop.Minimum = float64(defaultCfg.MinForecastInterval)
	prop.Default = defaultCfg.DefaultForecastInterval
	prop.ReadOnly = false

	prop = tdoc.AddProperty("forecastUpdated", "Weather Forecast Updated",
		"Time the weather forecast was last updated",
		vocab.WoTDataTypeDateTime)
	prop.ReadOnly = true

	prop = tdoc.AddProperty("latitude", "Latitude",
		"Location latitude",
		vocab.WoTDataTypeNumber)
	prop.ReadOnly = false
	prop = tdoc.AddProperty("longitude", "Longitude",
		"Location longitude",
		vocab.WoTDataTypeNumber)
	prop.ReadOnly = false

	prop = tdoc.AddProperty("locationName", "Name",
		"Location Name",
		vocab.WoTDataTypeString)
	prop.ReadOnly = false

	prop = tdoc.AddProperty("provider", "Weather Provider",
		"Override the default weather provider",
		vocab.WoTDataTypeString)
	prop.Enum = []any{providers.OpenMeteoProviderID}

	// this event contains the weather forecast
	ev := tdoc.AddEvent(EventNameCurrentWeather, "Current Weather", "Current weather",
		&td.DataSchema{
			Title:    "Weather Data",
			Format:   "",
			ReadOnly: true,
			Type:     wot.DataTypeObject,
			Properties: map[string]*td.DataSchema{
				"pressureMsl": {
					Title:       "Sea Level Pressure",
					Description: "Sea level equivalent pressure at " + tdoc.Title,
					Unit:        vocab.UnitHectoPascal, // default hpa (=mbar)
					Type:        wot.DataTypeNumber,
				},
				"pressureSurface": {
					Title:       "Surface Pressure",
					Description: "Surface pressure at " + tdoc.Title,
					Unit:        vocab.UnitHectoPascal,
					Type:        wot.DataTypeNumber,
				},
				"precipitation": {
					Title:       "Precipitation",
					Description: "Precipitation",
					Unit:        vocab.UnitMilliMeter,
					Type:        wot.DataTypeInteger,
				},
				"rain": {
					Title:       "Rain",
					Description: "Rain",
					Unit:        vocab.UnitMilliMeter,
					Type:        wot.DataTypeNumber,
				},
				"humidity": {
					Title: "Relative Humidity",
					Unit:  vocab.UnitPercent,
					Type:  wot.DataTypeInteger,
				},
				"temperature": {
					Title:       "Temperature",
					Description: "Temperature at 10 meter",
					Unit:        vocab.UnitCelcius,
					Type:        wot.DataTypeNumber,
				},
				"windDirection": {
					Title:       "Wind Direction",
					Description: "Wind heading at 10 meter in 0-359 degree",
					Unit:        vocab.UnitDegree,
					Type:        wot.DataTypeInteger,
				},
				"windSpeed": {
					Title: "Wind Speed",
					Unit:  vocab.UnitMeterPerSecond,
					Type:  wot.DataTypeNumber,
				},
			},
		})

	// this event contains the weather forecast
	ev = tdoc.AddEvent(EventNameForecastWeather, "Weather Forecast", "7 Day Weather Forecast",
		&td.DataSchema{
			Title: "Forecast Data",
			Type:  wot.DataTypeArray,
			ArrayItems: &td.DataSchema{
				Title: "Forecast",
				// todo: only add properties whose forecast is included
				Properties: map[string]*td.DataSchema{
					"precipitation": {
						Title: "Precipitation",
						Unit:  vocab.UnitMilliMeter,
						Type:  wot.DataTypeInteger,
					},
					"rain": {
						Title: "Rain",
						Unit:  vocab.UnitMilliMeter,
						Type:  wot.DataTypeNumber,
					},
					"relativeHumidity": {
						Title: "Relative Humidity",
						Unit:  vocab.UnitPercent,
						Type:  wot.DataTypeInteger,
					},
					"time": {
						Title: "Forecast Time",
						Type:  wot.DataTypeDateTime,
					},
					"temperature": {
						Title: "Temperature",
						Unit:  vocab.UnitCelcius,
						Type:  wot.DataTypeNumber,
					},
					"windDirection": {
						Title: "Wind Direction",
						Unit:  vocab.UnitDegree,
						Type:  wot.DataTypeInteger,
					},
					"windSpeed": {
						Title: "Wind Speed",
						Unit:  vocab.UnitMeterPerSecond,
						Type:  wot.DataTypeNumber,
					},
				},
			},
		})
	_ = ev

	return tdoc
}

// PublishBindingTD publishes the TD of the binding itself
func PublishBindingTD(ag *messaging.Agent) error {
	tdoc := CreateBindingTD(ag.GetClientID())
	err := ag.PubTD(tdoc)
	return err
}

// PublishLocationTDs publishes the TD of all locationStore
func PublishLocationTDs(ag *messaging.Agent, defaultCfg *config.WeatherConfig, locationStore *LocationStore) (err error) {
	locationStore.ForEach(func(loc config.WeatherLocation) {
		err2 := PublishLocationTD(ag, defaultCfg, loc)
		if err2 != nil {
			err = err2
		}
	})
	return err
}

// PublishLocationTD publishes the TD of the given location
func PublishLocationTD(ag *messaging.Agent, defaultCfg *config.WeatherConfig, loc config.WeatherLocation) error {
	tdoc := CreateTDOfLocation(defaultCfg, &loc)
	err := ag.PubTD(tdoc)
	return err
}
