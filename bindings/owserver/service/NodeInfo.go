package service

import (
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hivekit/go/wot/vocab"
)

// define 1-wire node information

type DeviceFamilyInfo struct {
	Code        string
	DeviceType  string // HiveOT device type
	DeviceName  string
	Description string
}

// TODO: more device info based on family
var deviceFamilyMap = map[string]DeviceFamilyInfo{
	"": {
		Code:        "",
		DeviceType:  vocab.DeviceNetGateway, // Gateway has no family
		DeviceName:  "Gateway",
		Description: "",
	},
}

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
// Todo: get from config file so it is easy to update
var deviceTypeMap = map[string]string{
	"":   vocab.DeviceNetGateway,        // Gateway has no family
	"01": "serialNumber",                // 2401,2411 (1990A): Silicon Serial Number
	"02": "securityKey",                 // 1425 (1991): multikey 1153bit secure
	"04": vocab.DeviceTime,              // 2404 (1994): econoram time chip
	"05": vocab.DeviceActuatorSwitch,    // 2405: Addressable Switch
	"06": vocab.DeviceComputerMemory,    // (1993) 4K memory ibutton
	"08": vocab.DeviceComputerMemory,    // (1992) 1K memory ibutton
	"0A": vocab.DeviceComputerMemory,    // (1995) 16K memory ibutton
	"0C": vocab.DeviceComputerMemory,    // (1996) 64K memory ibutton
	"10": vocab.DeviceSensorThermometer, // 18S20: high precision digital thermometer
	"12": vocab.DeviceActuatorSwitch,    // 2406:  dual addressable switch plus 1k memory
	"14": vocab.DeviceComputerMemory,    // 2430A (1971): 256 EEPROM
	"1A": vocab.DeviceComputerMemory,    // (1963L) 4K Monetary
	"1C": vocab.DeviceComputerMemory,    // 28E04-100: 4K EEPROM with PIO
	"1D": vocab.DeviceComputerMemory,    // 2423:  4K RAM with counter
	"1F": "coupler",                     // 2409:  Microlan coupler?
	"20": "adconverter",                 // 2450:  Quad A/D convert
	"21": vocab.DeviceSensorThermometer, // 1921:  Thermochron iButton device
	"22": vocab.DeviceSensorThermometer, // 1822:  Econo digital thermometer
	"24": vocab.DeviceTime,              // 2415:  time chip
	"26": vocab.DeviceBatteryMonitor,    // 2438:  smart battery monitor
	"27": vocab.DeviceTime,              // 2417:  time chip with interrupt
	"28": vocab.DeviceSensorThermometer, // 18B20: programmable resolution digital thermometer
	"29": vocab.DeviceActuatorSwitch,    // 2408:  8-channel addressable switch
	"2C": vocab.DeviceSensor,            // 2890:  digital potentiometer"
	"2D": vocab.DeviceComputerMemory,    // 2431:  1k eeprom
	"2E": vocab.DeviceBatteryMonitor,    // 2770:  battery monitor and charge controller
	"30": vocab.DeviceBatteryMonitor,    // 2760, 2761, 2762:  high-precision li+ battery monitor
	"31": vocab.DeviceBatteryMonitor,    // 2720: efficient addressable single-cell rechargable lithium protection ic
	"33": vocab.DeviceComputerMemory,    // 2432 (1961S): 1k protected eeprom with SHA-1
	"36": vocab.DeviceSensor,            // 2740: high precision coulomb counter
	"37": vocab.DeviceComputerMemory,    // (1977): OWServerPassword protected 32k eeprom
	"3B": vocab.DeviceSensorThermometer, // DS1825: programmable digital thermometer (https://www.analog.com/media/en/technical-documentation/data-sheets/ds1825.pdf)
	"41": vocab.DeviceSensorThermometer, // 2422: Temperature Logger 8k mem
	"42": vocab.DeviceSensorThermometer, // DS28EA00: digital thermometer with PIO (https://www.analog.com/media/en/technical-documentation/data-sheets/ds28ea00.pdf)
	"51": vocab.DeviceIndicator,         // 2751: multi chemistry battery fuel gauge
	"84": vocab.DeviceTime,              // 2404S: dual port plus time
	//# EDS0068: Temperature, Humidity, Barometric Pressure and Light Sensor
	//https://www.embeddeddatasystems.com/assets/images/supportFiles/manuals/EN-UserMan%20%20OW-ENV%20Sensor%20v13.pdf
	"7E": vocab.DeviceSensorMulti,
}

// AttrConversion defines the conversion of 1-wire node to wot property or event affordance
type AttrConversion struct {
	DataType string
	// the amount of change that should trigger an event
	ChangeNotify float64
	Description  string
	Enum         []td.DataSchema
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
	"BarometricPressureHg": {Ignore: true},
	"BarometricPressureMb": {
		IsEvent: true, IsProp: true,
		Title:       "Pressure",
		Description: "Atmospheric pressure",
		VocabType:   vocab.PropEnvPressureSurface,
		DataType:    td.DataTypeNumber, Precision: 0, ChangeNotify: 3.0,
		Unit: vocab.UnitMilliBar,
	},
	"BarometricPressureMbHighAlarmValue": {
		IsProp: true, Ignore: false,
		Title:       "Pressure High Threshold",
		Description: "High atmospheric pressure alarm set value",
		//VocabType: vocab.PropAlarmConfig,
		DataType: td.DataTypeNumber, ChangeNotify: 1,
		Unit: vocab.UnitMilliBar,
	},
	"BarometricPressureMbHighAlarmState": {
		IsEvent: true, IsProp: true, Ignore: false,
		Title:       "Pressure High Alarm",
		Description: "High atmospheric pressure alarm status",
		VocabType:   vocab.PropAlarmStatus,
		DataType:    td.DataTypeBool, ChangeNotify: 1,
	},
	"BarometricPressureMbLowAlarmValue": {
		IsProp: true, Ignore: false,
		Title:       "Pressure Low Threshold",
		Description: "Low atmospheric pressure alarm set value",
		//VocabType: vocab.PropAlarmConfig,
		DataType: td.DataTypeNumber, ChangeNotify: 1,
		Unit: vocab.UnitMilliBar,
	},
	"BarometricPressureMbLowAlarmState": {
		IsEvent: true, IsProp: true, Ignore: false,
		Title:       "Pressure Low Alarm",
		Description: "Low atmospheric pressure alarm status",
		VocabType:   vocab.PropAlarmStatus,
		DataType:    td.DataTypeBool, ChangeNotify: 1,
	},
	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
	"Channel": {
		IsProp: true, Ignore: false,
		Title: "Channel", Description: "OWServer connection port",
		DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"ClearAlarms": {
		IsActuator: true, Ignore: false,
		Title:    "Clear Alarms",
		DataType: td.DataTypeNone, // no data with this action
	},
	"Counter1": {
		IsProp: true, Ignore: true,
		Title: "Sample counter 1", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"Counter2": {
		IsProp: true, Ignore: true,
		Title: "Sample counter 2", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel1": {
		IsProp: true,
		Title:  "Data Errors Channel 1", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel2": {
		IsProp: true,
		Title:  "Data Errors Channel 2", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DataErrorsChannel3": {
		IsProp: true,
		Title:  "Data Errors Channel 3", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DateTime": {
		IsProp: true,
		Title:  "Device date/time", DataType: td.DataTypeDateTime,
		ChangeNotify: -1, // do not report changes to time
	},
	// gateway has 'DeviceName', nodes use 'Name'
	"DeviceName": {
		IsProp:    true,
		Title:     "Device Name",
		VocabType: vocab.PropDeviceTitle, DataType: td.DataTypeString, ChangeNotify: 1,
	},
	"DevicesConnectedChannel1": {
		IsProp: true,
		Title:  "Nr Devices on Channel 1", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DevicesConnectedChannel2": {
		IsProp: true,
		Title:  "Nr Devices on Channel 2", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DevicesConnectedChannel3": {
		IsProp: true,
		Title:  "Nr Devices on Channel 3", DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"DewPoint": {
		IsProp: true, Ignore: true,
		Title:     "Dew point",
		VocabType: vocab.PropEnvDewpoint,
		DataType:  td.DataTypeNumber, Precision: 1, ChangeNotify: 1.0,
		Unit: vocab.UnitCelcius, //tbd
	},
	"Family": {
		IsProp:      true,
		Title:       "Family",
		Description: "Family number as read from 1-wire device",
		DataType:    td.DataTypeString,
	},
	"Health": {
		IsProp: true,
		Title:  "Health", Description: "Sensor connection health. Range 0-7",
		DataType: td.DataTypeNumber, Precision: 0, ChangeNotify: 1,
	},
	"HeatIndex": {Ignore: true},
	"HostName": {
		IsProp:    true,
		VocabType: vocab.PropNetHostname,
		Title:     "Hostname",
		DataType:  td.DataTypeString, ChangeNotify: 1,
	},
	//"HeatIndex":  {AtType: vocab.PropEnvHeatindex, Title: "Heat Index", DataType: td.DataTypeNumber, Precision: 1},
	"Humidity": {
		IsEvent: true, IsProp: true,
		Title: "Humidity", Description: "Relative humidity in %",
		VocabType: vocab.PropEnvHumidity,
		DataType:  td.DataTypeNumber, Precision: 0, ChangeNotify: 3.0,
		Unit: vocab.UnitPercent,
	},
	"Humidex": {
		IsEvent: true,
		Title:   "Humidex", Description: "Feels-like temperature",
		VocabType: vocab.PropEnvHumidex,
		DataType:  td.DataTypeNumber, Precision: 0, ChangeNotify: 1.0,
		Unit: vocab.UnitCelcius,
	},
	"HumidityHighAlarmState": {
		IsProp: true, IsEvent: true, Ignore: true,
		Title:     "Humidity High Alarm",
		VocabType: vocab.PropAlarmStatus,
		DataType:  td.DataTypeBool,
	},
	"HumidityLowAlarmState": {
		IsProp: true, IsEvent: true, Ignore: true,
		Title:     "Humidity Low Alarm",
		VocabType: vocab.PropAlarmStatus,
		DataType:  td.DataTypeBool,
	},
	"LED": {
		IsProp:   true,
		Title:    "LED on/off State",
		DataType: td.DataTypeBool, // On/Off
	},
	"LEDFunction": {
		IsProp:      true,
		Title:       "LED function",
		Description: "LED on/off behavior on alarm or manual command",
		DataType:    td.DataTypeInteger, ChangeNotify: 1,
		Enum: []td.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"LEDState": {
		IsProp: true, Ignore: true,
		Title:    "LED switch",
		DataType: td.DataTypeBool,
	},
	"Light": {
		IsEvent:   true,
		Title:     "Luminance",
		VocabType: vocab.PropEnvLuminance,
		DataType:  td.DataTypeNumber, Precision: 0, ChangeNotify: 20.0,
	},
	"MACAddress": {
		IsProp:    true,
		Title:     "MAC Address",
		VocabType: vocab.PropNetMAC, DataType: td.DataTypeString,
	},
	"Manufacturer": {
		IsProp:    true,
		Title:     "Manufacturer",
		VocabType: vocab.PropDeviceMake, DataType: td.DataTypeString,
	},
	"Model": {
		IsProp:    true,
		Title:     "Model",
		VocabType: vocab.PropDeviceModel, DataType: td.DataTypeString,
	},
	"Name": {
		IsProp: true,
		// the name seems to hold the model number
		Title:     "Device model",
		VocabType: vocab.PropDeviceModel, DataType: td.DataTypeString,
	},
	"PollCount": {
		IsProp: true, Ignore: true,
		Title:    "Poll Count",
		DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"PowerSource": {
		IsProp:   true,
		Title:    "Power Source",
		DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"PrimaryValue": {
		IsProp: false, Ignore: true,
		Title:    "Primary Value",
		DataType: td.DataTypeString,
	},
	"Resolution": {
		IsProp:   true,
		Title:    "Resolution",
		DataType: td.DataTypeInteger, Unit: "bits", ChangeNotify: 1,
	},
	"RawData": {
		IsProp: true, Ignore: true,
		Title:    "Raw Data",
		DataType: td.DataTypeString,
	},
	"Relay": { // relay is read-only status, RelayState is input control
		IsProp:      true,
		Title:       "Relay status",
		Description: "Current alarm status of the relay (function 0 and 1)",
		// the internal data type is integer 0=off, 1=on
		// how to present as a switch?
		DataType: td.DataTypeInteger, ChangeNotify: 1,
		VocabType: vocab.PropAlarmStatus,
		Enum: []td.DataSchema{
			{Const: "0", Title: "Off"},
			{Const: "1", Title: "On"},
		},
	},
	"RelayFunction": {
		IsProp:      true,
		VocabType:   vocab.PropStatusOnOff,
		Title:       "Relay function",
		Description: "Relay on/off behavior on alarm or manual command",
		DataType:    td.DataTypeInteger, ChangeNotify: 1,
		Enum: []td.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"RelayState": {
		IsProp: true, IsActuator: false,
		Title:       "Relay control",
		Description: "On/off relay control (Relay Function 2)",
		DataType:    td.DataTypeInteger, ChangeNotify: 1,
		VocabType: vocab.ActionSwitchOnOff,
		Enum: []td.DataSchema{
			{Const: "0", Title: "Off"},
			{Const: "1", Title: "On"},
		},
	},
	"ROMId": {
		IsProp:   true,
		Title:    "Device ROM ID",
		DataType: td.DataTypeString,
	},
	"Temperature": {
		IsEvent: true, IsProp: true,
		Title:     "Temperature",
		VocabType: vocab.PropEnvTemperature,
		// prevent a lot of events if temperature is on the edge of two values
		DataType: td.DataTypeNumber, Precision: 1, ChangeNotify: 0.1,
		Unit: vocab.UnitCelcius,
	},
	"TemperatureHighAlarmValue": {
		IsProp: true, Ignore: false,
		Title:       "Temperature High Threshold",
		Description: "High temperature alarm set value",
		//VocabType: vocab.PropAlarmConfig,
		DataType: td.DataTypeNumber, Precision: 1, ChangeNotify: 0.1,
		Unit: vocab.UnitCelcius,
	},
	"TemperatureHighAlarmState": {
		IsProp: true, IsEvent: true, Ignore: false,
		Title:     "Temperature High Alarm",
		VocabType: vocab.PropAlarmStatus,
		DataType:  td.DataTypeBool,
	},
	"TemperatureLowAlarmValue": {
		IsProp: true, Ignore: false,
		Title:       "Temperature Low Threshold",
		Description: "Low temperature alarm set value",
		//VocabType: vocab.PropAlarmConfig,
		DataType: td.DataTypeNumber, Precision: 1, ChangeNotify: 0.1,
		Unit: vocab.UnitCelcius,
	},
	"TemperatureLowAlarmState": {
		IsProp: true, IsEvent: true, Ignore: false,
		Title:     "Temperature Low Alarm",
		VocabType: vocab.PropAlarmStatus,
		DataType:  td.DataTypeBool,
	},
	"UserByte1": {
		IsProp:   true, //Ignore: true,
		Title:    "User Byte 1",
		DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"UserByte2": {
		IsProp:   true, //Ignore: true,
		Title:    "User Byte 2",
		DataType: td.DataTypeInteger, ChangeNotify: 1,
	},
	"Version": {
		IsProp:    true,
		VocabType: vocab.PropDeviceFirmwareVersion,
		Title:     "Firmware version",
		DataType:  td.DataTypeString,
	},
	"VoltageChannel1": {
		IsProp:    true,
		Title:     "Voltage Channel 1",
		DataType:  td.DataTypeNumber,
		VocabType: vocab.PropElectricVoltage,
		Unit:      vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltageChannel2": {
		IsProp:    true,
		Title:     "Voltage Channel 2",
		DataType:  td.DataTypeNumber,
		VocabType: vocab.PropElectricVoltage,
		Unit:      vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltageChannel3": {
		IsProp:    true,
		Title:     "Voltage Channel 3",
		DataType:  td.DataTypeNumber,
		VocabType: vocab.PropElectricVoltage,
		Unit:      vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
	"VoltagePower": {
		IsProp:    true,
		Title:     "Voltage powerline",
		DataType:  td.DataTypeNumber,
		VocabType: vocab.PropElectricVoltage,
		Unit:      vocab.UnitVolt, Precision: 1, ChangeNotify: 0.1,
	},
}

// UnitNameVocab maps OWServer unit names to IoT vocabulary
var UnitNameVocab = map[string]string{
	"PercentRelativeHumidity": vocab.UnitPercent,
	"Millibars":               vocab.UnitMilliBar,
	"Centigrade":              vocab.UnitCelcius,
	"Fahrenheit":              vocab.UnitFahrenheit,
	"InchesOfMercury":         vocab.UnitMercury,
	"Lux":                     vocab.UnitLux,
	"Volt":                    vocab.UnitVolt,
}
