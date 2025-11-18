package service

import (
	"github.com/hiveot/hivekit/go/agent"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
)

// read-only properties
const PropNameCurrentUpdated = "currentUpdated"
const PropNameHourlyUpdated = "hourlyUpdated"

// configuration properties
const PropNameCurrentEnabled = "currentEnabled"
const PropNameCurrentInterval = "currentInterval"
const PropNameWeatherProvider = "weatherProvider"
const PropNameHourlyEnabled = "hourlyEnabled"
const PropNameHourlyInterval = "hourlyInterval"
const PropNameUnitsWindSpeed = "windSpeedUnits"

// actions
const ActionNameAddLocation = "addLocation"
const ActionNameRemoveLocation = "removedLocation"

// CreateBindingTD creates a Thing TD of this service
// This binding exposes the TD of itself.
func CreateBindingTD(serviceID string) *td.TD {
	// the agent-ID is the thingID of the binding
	tdoc := td.NewTD(serviceID, "Weather binding", vocab.ThingService)
	tdoc.Description = "Binding for the weather service"

	// The defaults are defined in the config yaml.
	prop := tdoc.AddProperty(PropNameWeatherProvider, "Default Provider",
		"Name of the current default weather provider used",
		vocab.WoTDataTypeString)
	prop.ReadOnly = true
	prop.Enum = []any{providers.OpenMeteoProviderID}

	prop = tdoc.AddProperty(PropNameUnitsWindSpeed, "Wind Speed Units",
		"The units for wind speed",
		vocab.WoTDataTypeString)
	prop.Default = vocab.UnitMeterPerSecond
	prop.ReadOnly = true
	prop.DataSchema.SetEnumAsStrings([]string{vocab.UnitMeterPerSecond, vocab.UnitKilometerPerHour, vocab.UnitMilesPerHour})

	action := tdoc.AddAction(ActionNameAddLocation, "Add Location", "Add a new location", &td.DataSchema{
		Title:  "Location configuration",
		Type:   wot.DataTypeObject,
		Schema: "WeatherLocation",
	})
	action.Safe = false
	action.Idempotent = true

	action = tdoc.AddAction(ActionNameRemoveLocation, "Remove Location", "Remove a location", &td.DataSchema{
		Title: "Location ID",
		Type:  wot.DataTypeString,
	})
	action.Safe = false
	action.Idempotent = true

	return tdoc
}

// CreateTDOfLocation creates a Thing TD describing a weather location.
//
// TODO: should this be done by the provider? Different providers have different capabilities
func CreateTDOfLocation(defaultCfg *config.WeatherConfig, cfg *config.WeatherLocation) *td.TD {
	thingID := cfg.ID
	deviceType := vocab.ThingSensorEnvironment
	title := cfg.Name
	tdoc := td.NewTD(thingID, title, deviceType)
	tdoc.Description = "Current weather for " + cfg.Name

	// Attributes

	prop := tdoc.AddProperty(PropNameCurrentUpdated, "Current Weather Updated",
		"Time the current weather was last updated",
		vocab.WoTDataTypeDateTime)
	prop.ReadOnly = true

	prop = tdoc.AddProperty(PropNameWeatherProvider, "Weather Provider",
		"The weather provider for this location",
		vocab.WoTDataTypeString)
	prop.Enum = []any{providers.OpenMeteoProviderID}
	prop.Default = defaultCfg.DefaultProvider

	// Configuration

	prop = tdoc.AddProperty(PropNameCurrentEnabled, "Enable Current Weather",
		"Enable read the current weather",
		vocab.WoTDataTypeBool)
	prop.ReadOnly = false
	prop.Default = defaultCfg.DefaultCurrentEnabled

	prop = tdoc.AddProperty(PropNameCurrentInterval, "Current Weather Updates",
		"Interval in seconds the current weather is updated",
		vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitSecond
	prop.Minimum = float64(defaultCfg.MinCurrentInterval)
	prop.Default = defaultCfg.DefaultCurrentInterval
	prop.ReadOnly = false

	prop = tdoc.AddProperty(PropNameHourlyEnabled, "Enable Hourly Forecast",
		"Enable reading the weather forecast",
		vocab.WoTDataTypeBool)
	prop.Default = defaultCfg.DefaultHourlyEnabled
	prop.ReadOnly = false

	prop = tdoc.AddProperty(PropNameHourlyInterval, "Hourly Forecast Interval",
		"Interval to update the hourly forecast",
		vocab.WoTDataTypeInteger)
	prop.ReadOnly = false
	prop.Default = defaultCfg.DefaultHourlyForecastInterval
	prop.Unit = vocab.UnitSecond

	prop = tdoc.AddProperty(PropNameHourlyUpdated, "Hourly Forecast Updated time",
		"Time the hourly weather forecast was last updated",
		vocab.WoTDataTypeDateTime)
	prop.ReadOnly = true

	prop = tdoc.AddProperty(vocab.PropLocationLatitude, "Latitude",
		"Location latitude",
		vocab.WoTDataTypeNumber)
	prop.ReadOnly = false
	prop.AtType = vocab.PropLocationLatitude
	prop = tdoc.AddProperty(vocab.PropLocationLongitude, "Longitude",
		"Location longitude",
		vocab.WoTDataTypeNumber)
	prop.ReadOnly = false
	prop.AtType = vocab.PropLocationLongitude

	prop = tdoc.AddProperty(vocab.PropLocationName, "Name",
		"Location Name",
		vocab.WoTDataTypeString)
	prop.ReadOnly = false

	// Events with the current weather

	ev := tdoc.AddEvent(vocab.PropEnvHumidity, "Relative Humidity", "",
		&td.DataSchema{
			Unit:   vocab.UnitPercent,
			Type:   wot.DataTypeInteger,
			AtType: vocab.PropEnvHumidity,
		})

	ev = tdoc.AddEvent(vocab.PropEnvPrecipitation, "Precipitation",
		"Rain or snow precipiation",
		&td.DataSchema{
			Unit:   vocab.UnitMilliMeter,
			Type:   wot.DataTypeInteger,
			AtType: vocab.PropEnvPrecipitation,
		})

	ev = tdoc.AddEvent(vocab.PropEnvPressureSeaLevel, "Sea Level Pressure",
		"Sea level equivalent pressure at "+tdoc.Title,
		&td.DataSchema{
			Unit:   vocab.UnitHectoPascal, // default hpa (=mbar)
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvPressureSeaLevel,
		})
	ev = tdoc.AddEvent(vocab.PropEnvPressureSurface, "Surface Pressure",
		"Surface pressure at "+tdoc.Title,
		&td.DataSchema{
			Unit:   vocab.UnitHectoPascal,
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvPressureSurface,
		})

	ev = tdoc.AddEvent(vocab.PropEnvPrecipitationRain, "Rain", "Rainfall amount last hour",
		&td.DataSchema{
			Unit:   vocab.UnitMilliMeter,
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvPrecipitationRain,
		})

	ev = tdoc.AddEvent(vocab.PropEnvPrecipitationSnow, "Snow", "Snowfall amount last hour",
		&td.DataSchema{
			Unit:   vocab.UnitMilliMeter,
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvPrecipitationSnow,
		})

	ev = tdoc.AddEvent(vocab.PropEnvTemperature, "Temperature",
		"Temperature at 10 meter",
		&td.DataSchema{
			Unit:   vocab.UnitCelcius,
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvTemperature,
		})

	ev = tdoc.AddEvent(vocab.PropEnvWindHeading, "Wind Direction",
		"Wind heading at 10 meter in 0-359 degree",
		&td.DataSchema{
			Unit:   vocab.UnitDegree,
			Type:   wot.DataTypeInteger,
			AtType: vocab.PropEnvWindHeading,
		})

	ev = tdoc.AddEvent(vocab.PropEnvWindGusts, "Wind Gusts", "",
		&td.DataSchema{
			//Unit: vocab.UnitMeterPerSecond,
			Unit:   vocab.UnitKilometerPerHour, // TODO: configurable
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvWindGusts,
		})

	ev = tdoc.AddEvent(vocab.PropEnvWindSpeed, "Wind Speed", "",
		&td.DataSchema{
			//Unit: vocab.UnitMeterPerSecond,
			Unit:   vocab.UnitKilometerPerHour, // TODO: configurable
			Type:   wot.DataTypeNumber,
			AtType: vocab.PropEnvWindSpeed,
		})

	_ = ev

	return tdoc
}

// PublishBindingTD publishes the TD of the binding itself
func PublishBindingTD(ag *agent.Agent, cfg *config.WeatherConfig) error {
	thingID := ag.GetClientID()
	tdoc := CreateBindingTD(thingID)
	err := ag.UpdateThing(tdoc)
	if err == nil {
		err = PublishBindingProperties(ag, thingID, cfg)
	}
	return err
}

// PublishLocationTDs publishes the TD of all locationStore and current properties
func PublishLocationTDs(ag *agent.Agent, defaultCfg *config.WeatherConfig, locationStore *LocationStore) (err error) {
	locationStore.ForEach(func(loc config.WeatherLocation) {
		err2 := PublishLocationTD(ag, defaultCfg, loc)
		if err2 != nil {
			err = err2
		}
	})
	return err
}

// PublishLocationTD publishes the TD of the given location and its current values
func PublishLocationTD(ag *agent.Agent, defaultCfg *config.WeatherConfig, loc config.WeatherLocation) error {
	tdoc := CreateTDOfLocation(defaultCfg, &loc)
	err := ag.UpdateThing(tdoc)
	if err == nil {
		err = PublishLocationProperties(ag, loc.ID, loc)
	}
	return err
}
