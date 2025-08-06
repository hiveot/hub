package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
)

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
	tdoc.Description = "Current weather for " + cfg.Name

	// Attributes

	prop := tdoc.AddProperty("currentUpdated", "Current Weather Updated",
		"Time the current weather was last updated",
		vocab.WoTDataTypeDateTime)
	prop.ReadOnly = true

	//prop = tdoc.AddProperty("provider", "Weather Provider",
	//	"The weather provider for this location",
	//	vocab.WoTDataTypeString)
	//prop.Enum = []any{providers.OpenMeteoProviderID}

	// Configuration

	prop = tdoc.AddProperty("currentEnabled", "Enable Current Weather",
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

	prop = tdoc.AddProperty("units-wind-speed", "Wind Speed Units",
		"The units for wind speed",
		vocab.WoTDataTypeString)
	prop.Default = vocab.UnitMeterPerSecond
	prop.ReadOnly = false
	prop.DataSchema.SetEnumAsStrings([]string{vocab.UnitMeterPerSecond, vocab.UnitKilometerPerHour, vocab.UnitMilesPerHour})

	// Events with the current weather

	ev := tdoc.AddEvent("humidity", "Relative Humidity", "",
		&td.DataSchema{
			Unit: vocab.UnitPercent,
			Type: wot.DataTypeInteger,
		})

	ev = tdoc.AddEvent("precipitation", "Precipitation",
		"Rain or snow precipiation",
		&td.DataSchema{
			Unit: vocab.UnitMilliMeter,
			Type: wot.DataTypeInteger,
		})

	ev = tdoc.AddEvent("pressureMsl", "Sea Level Pressure",
		"Sea level equivalent pressure at "+tdoc.Title,
		&td.DataSchema{
			Unit: vocab.UnitHectoPascal, // default hpa (=mbar)
			Type: wot.DataTypeNumber,
		})
	ev = tdoc.AddEvent("pressureSurface", "Surface Pressure",
		"Surface pressure at "+tdoc.Title,
		&td.DataSchema{
			Unit: vocab.UnitHectoPascal,
			Type: wot.DataTypeNumber,
		})

	ev = tdoc.AddEvent("rain", "Rain", "Rainfall amount",
		&td.DataSchema{
			Unit: vocab.UnitMilliMeter,
			Type: wot.DataTypeNumber,
		})

	ev = tdoc.AddEvent("temperature", "Temperature",
		"Temperature at 10 meter",
		&td.DataSchema{
			Unit: vocab.UnitCelcius,
			Type: wot.DataTypeNumber,
		})

	ev = tdoc.AddEvent("windDirection", "Wind Direction",
		"Wind heading at 10 meter in 0-359 degree",
		&td.DataSchema{
			Unit: vocab.UnitDegree,
			Type: wot.DataTypeInteger,
		})

	ev = tdoc.AddEvent("windSpeed", "Wind Speed", "",
		&td.DataSchema{
			Unit: vocab.UnitMeterPerSecond,
			Type: wot.DataTypeNumber,
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
