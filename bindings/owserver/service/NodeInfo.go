package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	thing "github.com/hiveot/hub/lib/things"
)

// define 1-wire node information

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
// Todo: get from config file so it is easy to update
var deviceTypeMap = map[string]string{
	"":   vocab.ThingNetGateway,           // Gateway has no family
	"01": "serialNumber",                  // 2401,2411 (1990A): Silicon Serial Number
	"02": "securityKey",                   // 1425 (1991): multikey 1153bit secure
	"04": vocab.ThingDeviceTime,           // 2404 (1994): econoram time chip
	"05": vocab.ThingActuatorSwitch,       // 2405: Addressable Switch
	"06": vocab.ThingComputerMemory,       // (1993) 4K memory ibutton
	"08": vocab.ThingComputerMemory,       // (1992) 1K memory ibutton
	"0A": vocab.ThingComputerMemory,       // (1995) 16K memory ibutton
	"0C": vocab.ThingComputerMemory,       // (1996) 64K memory ibutton
	"10": vocab.ThingSensorThermometer,    // 18S20: high precision digital thermometer
	"12": vocab.ThingActuatorSwitch,       // 2406:  dual addressable switch plus 1k memory
	"14": vocab.ThingComputerMemory,       // 2430A (1971): 256 EEPROM
	"1A": vocab.ThingComputerMemory,       // (1963L) 4K Monetary
	"1C": vocab.ThingComputerMemory,       // 28E04-100: 4K EEPROM with PIO
	"1D": vocab.ThingComputerMemory,       // 2423:  4K RAM with counter
	"1F": "coupler",                       // 2409:  Microlan coupler?
	"20": "adconverter",                   // 2450:  Quad A/D convert
	"21": vocab.ThingSensorThermometer,    // 1921:  Thermochron iButton device
	"22": vocab.ThingSensorThermometer,    // 1822:  Econo digital thermometer
	"24": vocab.ThingDeviceTime,           // 2415:  time chip
	"26": vocab.ThingDeviceBatteryMonitor, // 2438:  smart battery monitor
	"27": vocab.ThingDeviceTime,           // 2417:  time chip with interrupt
	"28": vocab.ThingSensorThermometer,    // 18B20: programmable resolution digital thermometer
	"29": vocab.ThingActuatorSwitch,       // 2408:  8-channel addressable switch
	"2C": vocab.ThingSensor,               // 2890:  digital potentiometer"
	"2D": vocab.ThingComputerMemory,       // 2431:  1k eeprom
	"2E": vocab.ThingDeviceBatteryMonitor, // 2770:  battery monitor and charge controller
	"30": vocab.ThingDeviceBatteryMonitor, // 2760, 2761, 2762:  high-precision li+ battery monitor
	"31": vocab.ThingDeviceBatteryMonitor, // 2720: efficient addressable single-cell rechargable lithium protection ic
	"33": vocab.ThingComputerMemory,       // 2432 (1961S): 1k protected eeprom with SHA-1
	"36": vocab.ThingSensor,               // 2740: high precision coulomb counter
	"37": vocab.ThingComputerMemory,       // (1977): OWServerPassword protected 32k eeprom
	"3B": vocab.ThingSensorThermometer,    // DS1825: programmable digital thermometer (https://www.analog.com/media/en/technical-documentation/data-sheets/ds1825.pdf)
	"41": vocab.ThingSensorThermometer,    // 2422: Temperature Logger 8k mem
	"42": vocab.ThingSensorThermometer,    // DS28EA00: digital thermometer with PIO (https://www.analog.com/media/en/technical-documentation/data-sheets/ds28ea00.pdf)
	"51": vocab.ThingDeviceIndicator,      // 2751: multi chemistry battery fuel gauge
	"84": vocab.ThingDeviceTime,           // 2404S: dual port plus time
	//# EDS0068: Temperature, Humidity, Barometric Pressure and Light Sensor
	//https://www.embeddeddatasystems.com/assets/images/supportFiles/manuals/EN-UserMan%20%20OW-ENV%20Sensor%20v13.pdf
	"7E": vocab.ThingSensorMulti,
}

// AttrConversion defines the conversion of 1-wire node to wot property or event affordance
type AttrConversion struct {
	DataType string
	// the amount of change that should trigger an event
	ChangeNotify float64
	Enum         []thing.DataSchema
	// ignore the attribute
	Ignore bool
	// IsActuator defines the attribute as an action
	IsActuator bool
	// send changes as event
	IsEvent bool
	// include changes in property update
	IsProp bool
	// number of Precision accuracy for this value
	Precision int
	Title     string
	Unit      string // unit to use if provided
	VocabType string // sensor type from vocabulary
}

// AttrConfig defines known property/configuration attribute conversion
var AttrConfig = map[string]AttrConversion{
	//"BarometricPressureHg":               vocab.PropNameAtmosphericPressure,                                                                                  // unit Hg
	"BarometricPressureHg": {IsEvent: true, Ignore: true,
		Title:     "Atmospheric Pressure in Hg",
		VocabType: vocab.PropEnvBarometer,
		DataType:  vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 3.0,
		Unit: vocab.UnitMercury,
	},
	"BarometricPressureMb": {IsEvent: true,
		Title:     "Atmospheric Pressure",
		VocabType: vocab.PropEnvBarometer,
		DataType:  vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 3.0,
		Unit: vocab.UnitMillibar,
	},
	"BarometricPressureMbHighAlarmState": {IsEvent: true,
		Title:    "Pressure High Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	"BarometricPressureMbLowAlarmState": {IsEvent: true,
		Title:    "Pressure Low Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
	"ClearAlarms": {IsActuator: true,
		Title:    "Clear Alarms",
		DataType: vocab.WoTDataTypeNone, // no data with this action
	},
	"Counter1": {IsProp: true, Ignore: true,
		Title: "Sample counter 1", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"Counter2": {IsProp: true, Ignore: true,
		Title: "Sample counter 2", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel1": {IsProp: true,
		Title: "Data Errors Channel 1", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel2": {IsProp: true,
		Title: "Data Errors Channel 2", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel3": {IsProp: true,
		Title: "Data Errors Channel 3", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DateTime": {IsProp: true, Ignore: true,
		Title: "Device date/time", DataType: vocab.WoTDataTypeDateTime,
	},
	"DevicesConnectedChannel1": {IsProp: true,
		Title: "Nr Devices on Channel 1", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DevicesConnectedChannel2": {IsProp: true,
		Title: "Nr Devices on Channel 2", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DevicesConnectedChannel3": {IsProp: true,
		Title: "Nr Devices on Channel 3", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"DewPoint": {
		Title:     "Dew point",
		VocabType: vocab.PropEnvDewpoint,
		DataType:  vocab.WoTDataTypeNumber, Precision: 1, ChangeNotify: 1.0,
		Unit: vocab.UnitCelcius, //tbd
	},
	"Health": {IsEvent: true,
		Title:    "Health 0-7",
		DataType: vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 1,
	},
	"HeatIndex": {Ignore: true},
	"HostName": {IsProp: true,
		VocabType: vocab.PropNetHostname,
		Title:     "Hostname",
		DataType:  vocab.WoTDataTypeString,
	},
	//"HeatIndex":  {VocabType: vocab.PropEnvHeatindex, Title: "Heat Index", DataType: vocab.WoTDataTypeNumber, Precision: 1},
	"Humidity": {IsEvent: true,
		Title:     "Humidity",
		VocabType: vocab.PropEnvHumidity,
		DataType:  vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 3.0,
		Unit: vocab.UnitPercent,
	},
	"Humidex": {IsEvent: true,
		Title:     "Humidex",
		VocabType: vocab.PropEnvHumidex,
		DataType:  vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 1.0,
		Unit: vocab.UnitCelcius,
	},
	"HumidityHighAlarmState": {IsEvent: true,
		Title:    "Humidity High Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	"HumidityLowAlarmState": {IsEvent: true,
		Title:    "Humidity Low Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	"LED": {IsProp: true,
		Title: "LED control", DataType: vocab.WoTDataTypeString,
	},
	"LEDFunction": {IsProp: true,
		Title: "LED function control", DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
		Enum: []thing.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"Light": {IsEvent: true,
		Title:     "Luminance",
		VocabType: vocab.PropEnvLuminance,
		DataType:  vocab.WoTDataTypeNumber, Precision: 0, ChangeNotify: 30.0,
	},
	"Manufacturer": {IsProp: true,
		Title:     "Manufacturer",
		VocabType: vocab.PropDeviceMake, DataType: vocab.WoTDataTypeString,
	},
	"Model": {IsProp: true,
		Title: "Model", VocabType: vocab.PropDeviceModel, DataType: vocab.WoTDataTypeString,
	},
	"Name": {IsProp: true,
		// the name seems to hold the model number
		Title:     "Device name",
		VocabType: vocab.PropDeviceModel, DataType: vocab.WoTDataTypeString,
	},
	"PollCount": {IsProp: true, Ignore: true,
		Title:    "Poll Count",
		DataType: vocab.WoTDataTypeInteger, ChangeNotify: 1,
	},
	"PrimaryValue": {IsProp: false, Ignore: true,
		Title:    "Primary Value",
		DataType: vocab.WoTDataTypeString,
	},
	"Resolution": {IsProp: true,
		Title:    "Resolution",
		DataType: vocab.WoTDataTypeInteger, Unit: "bits", ChangeNotify: 1,
	},
	"RawData": {IsProp: true, Ignore: true,
		Title:    "Raw Data",
		DataType: vocab.WoTDataTypeString,
	},
	"Relay": {IsActuator: true,
		Title:    "Relay control",
		DataType: vocab.WoTDataTypeBool, VocabType: vocab.ActionSwitchOff,
	},
	"RelayFunction": {IsProp: true,
		VocabType: vocab.PropStatusOnOff,
		Title:     "Relay function control",
		DataType:  vocab.WoTDataTypeInteger, ChangeNotify: 1,
		Enum: []thing.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"Temperature": {IsEvent: true,
		Title:     "Temperature",
		VocabType: vocab.PropEnvTemperature,
		DataType:  vocab.WoTDataTypeNumber, Precision: 1, ChangeNotify: 0.1,
		Unit: vocab.UnitCelcius,
	},
	"TemperatureHighAlarmState": {IsEvent: true,
		Title:    "Temperature High Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	"TemperatureLowAlarmState": {IsEvent: true,
		Title:    "Temperature Low Alarm",
		DataType: vocab.WoTDataTypeBool,
	},
	"Version": {IsProp: true,
		VocabType: vocab.PropDeviceFirmwareVersion,
		Title:     "Firmware version",
		DataType:  vocab.WoTDataTypeString,
	},
	"VoltageChannel1": {IsProp: true,
		Title:    "Voltage Channel 1",
		DataType: vocab.WoTDataTypeNumber,
		Unit:     vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltageChannel2": {IsProp: true,
		Title:    "Voltage Channel 2",
		DataType: vocab.WoTDataTypeNumber,
		Unit:     vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltageChannel3": {IsProp: true,
		Title:    "Voltage Channel 3",
		DataType: vocab.WoTDataTypeNumber,
		Unit:     vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltagePower": {IsProp: true,
		Title:    "Voltage powerline",
		DataType: vocab.WoTDataTypeNumber,
		Unit:     vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
}

// UnitNameVocab maps OWServer unit names to IoT vocabulary
var UnitNameVocab = map[string]string{
	"PercentRelativeHumidity": vocab.UnitPercent,
	"Millibars":               vocab.UnitMillibar,
	"Centigrade":              vocab.UnitCelcius,
	"Fahrenheit":              vocab.UnitFahrenheit,
	"InchesOfMercury":         vocab.UnitMercury,
	"Lux":                     vocab.UnitLux,
	"Volt":                    vocab.UnitVolt,
}
