package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	thing "github.com/hiveot/hub/lib/things"
)

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

// ActuatorTypeVocab maps OWServer names to IoT vocabulary
var ActuatorTypeVocab = map[string]struct {
	VocabType string // sensor type from vocabulary
	Title     string
	DataType  string
}{
	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
	"ClearAlarms": {Title: "Clear Alarms"},
	"Relay":       {VocabType: vocab.ActionSwitchOff, Title: "Relay", DataType: vocab.WoTDataTypeBool},
}

// PropAttrVocab defines known property/configuration attributes
var PropAttrVocab = map[string]struct {
	VocabType string // sensor type from vocabulary
	Title     string
	DataType  string
	Decimals  int // number of Decimals accuracy for this value
	Enum      []thing.DataSchema
	Unit      string // unit to use if provided
}{
	"Counter1": {
		Title: "Sample counter 1", DataType: vocab.WoTDataTypeNumber},
	"Counter2": {
		Title: "Sample counter 2", DataType: vocab.WoTDataTypeNumber},
	"DataErrorsChannel1":       {Title: "Data Errors Channel 1", DataType: vocab.WoTDataTypeInteger},
	"DataErrorsChannel2":       {Title: "Data Errors Channel 2", DataType: vocab.WoTDataTypeInteger},
	"DataErrorsChannel3":       {Title: "Data Errors Channel 3", DataType: vocab.WoTDataTypeInteger},
	"DevicesConnectedChannel1": {Title: "Nr Devices on Channel 1", DataType: vocab.WoTDataTypeInteger},
	"DevicesConnectedChannel2": {Title: "Nr Devices on Channel 2", DataType: vocab.WoTDataTypeInteger},
	"DevicesConnectedChannel3": {Title: "Nr Devices on Channel 3", DataType: vocab.WoTDataTypeInteger},
	"HostName": {
		VocabType: vocab.PropNetHostname,
		Title:     "Hostname",
		DataType:  vocab.WoTDataTypeString,
	},
	"LED": {
		Title: "LED control", DataType: vocab.WoTDataTypeString},
	"LEDFunction": {
		Title: "LED function control", DataType: vocab.WoTDataTypeInteger,
		Enum: []thing.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"Manufacturer": {
		Title: "Manufacturer", VocabType: vocab.PropDeviceMake, DataType: vocab.WoTDataTypeString,
	},
	"Model": {
		Title: "Model", VocabType: vocab.PropDeviceModel, DataType: vocab.WoTDataTypeString,
	},
	"Name": {
		// the name seems to hold the model number
		Title: "Device name", VocabType: vocab.PropDeviceModel, DataType: vocab.WoTDataTypeString,
	},
	"Resolution": {
		Title: "Resolution", DataType: vocab.WoTDataTypeInteger, Unit: "bits",
	},
	"Relay": {
		Title: "Relay control", DataType: vocab.WoTDataTypeString,
	},
	"RelayFunction": {VocabType: vocab.PropStatusOnOff,
		Title: "Relay function control", DataType: vocab.WoTDataTypeInteger,
		Enum: []thing.DataSchema{
			{Const: "0", Title: "On with alarms, off with no alarms"},
			{Const: "1", Title: "On with alarms, Off with clear alarms command"},
			{Const: "2", Title: "On/Off under command"},
			{Const: "3", Title: "Always Off"},
		},
	},
	"Version": {VocabType: vocab.PropDeviceFirmwareVersion,
		Title: "Firmware version", DataType: vocab.WoTDataTypeString},
	"VoltageChannel1": {Title: "Voltage Channel 1",
		DataType: vocab.WoTDataTypeNumber, Unit: vocab.UnitVolt},
	"VoltageChannel2": {Title: "Voltage Channel 2",
		DataType: vocab.WoTDataTypeNumber, Unit: vocab.UnitVolt},
	"VoltageChannel3": {Title: "Voltage Channel 3",
		DataType: vocab.WoTDataTypeNumber, Unit: vocab.UnitVolt},
	"VoltagePower": {Title: "Voltage powerline",
		DataType: vocab.WoTDataTypeNumber, Unit: vocab.UnitVolt},
}

// PropIgnore ignore these properties
var PropIgnore = map[string]bool{
	"RawData": true,
}

// SensorAttrVocab maps OWServer sensor attribute names to IoT event and property vocabulary
var SensorAttrVocab = map[string]struct {
	VocabType string // sensor type from vocabulary
	Title     string
	DataType  string
	Decimals  int // number of Decimals accuracy for this value
}{
	//"BarometricPressureHg":               vocab.PropNameAtmosphericPressure,                                                                                  // unit Hg
	"BarometricPressureMb": {VocabType: vocab.PropEnvBarometer,
		Title: "Atmospheric Pressure", DataType: vocab.WoTDataTypeNumber, Decimals: 0,
	}, // unit Mb
	"BarometricPressureMbHighAlarmState": {
		VocabType: "", Title: "Pressure High Alarm", DataType: vocab.WoTDataTypeBool,
	},
	"BarometricPressureMbLowAlarmState": {
		VocabType: "", Title: "Pressure Low Alarm", DataType: vocab.WoTDataTypeBool,
	},
	"DewPoint": {VocabType: vocab.PropEnvDewpoint,
		Title: "Dew point", DataType: vocab.WoTDataTypeNumber, Decimals: 1,
	},
	"Health": {
		VocabType: "", Title: "Health 0-7", DataType: vocab.WoTDataTypeNumber,
	},
	//"HeatIndex":  {VocabType: vocab.PropEnvHeatindex, Title: "Heat Index", DataType: vocab.WoTDataTypeNumber, Decimals: 1},
	"Humidity": {
		VocabType: vocab.PropEnvHumidity,
		Title:     "Humidity", DataType: vocab.WoTDataTypeNumber, Decimals: 0,
	},
	"Humidex": {
		VocabType: vocab.PropEnvHumidex,
		Title:     "Humidex", DataType: vocab.WoTDataTypeNumber, Decimals: 0,
	},
	"HumidityHighAlarmState": {
		VocabType: "", Title: "Humidity High Alarm", DataType: vocab.WoTDataTypeBool,
	},
	"HumidityLowAlarmState": {
		VocabType: "", Title: "Humidity Low Alarm", DataType: vocab.WoTDataTypeBool,
	},
	"Light": {VocabType: vocab.PropEnvLuminance,
		Title: "Luminance", DataType: vocab.WoTDataTypeNumber, Decimals: 0,
	},
	"Temperature": {VocabType: vocab.PropEnvTemperature,
		Title: "Temperature", DataType: vocab.WoTDataTypeNumber, Decimals: 1,
	},
	"TemperatureHighAlarmState": {
		VocabType: "", Title: "Temperature High Alarm", DataType: vocab.WoTDataTypeBool,
	},
	"TemperatureLowAlarmState": {
		VocabType: "", Title: "Temperature Low Alarm", DataType: vocab.WoTDataTypeBool,
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

// CreateTDFromNode converts the 1-wire node into a TD that describes the node.
// - All attributes will be added as node properties
// - Writable non-sensors attributes are marked as writable configuration
// - Sensors are also added as events.
// - Writable sensors are also added as actions.
func CreateTDFromNode(node *eds.OneWireNode) (tdoc *thing.TD) {

	// Should we bother with the URI? In HiveOT things have pubsub addresses that include the ID. The ID is not the address.
	//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
	thingID := node.ROMId
	if thingID == "" {
		thingID = vocab.ThingNetGateway
	}
	deviceType := deviceTypeMap[node.Family]
	if deviceType == "" {
		// unknown device
		deviceType = vocab.ThingDevice
	}

	tdoc = thing.NewTD(thingID, node.Name, deviceType)
	tdoc.UpdateTitleDescription(node.Name, node.Description)

	// Map node attribute to Thing properties and events
	for attrID, attr := range node.Attr {
		sensorInfo, isSensor := SensorAttrVocab[attrID]
		actuatorInfo, isActuator := ActuatorTypeVocab[attrID]
		propInfo, isProp := PropAttrVocab[attrID]
		propIgnore, _ := PropIgnore[attrID]

		if isSensor {
			var evSchema *thing.DataSchema
			// only add data schema if the event carries a value
			if sensorInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				evSchema = &thing.DataSchema{
					Type:     sensorInfo.DataType,
					Unit:     unit,
					ReadOnly: true,
				}
			}
			// Only attributes with a vocab type will be sent as events
			tdoc.AddEvent(attrID, sensorInfo.VocabType, sensorInfo.Title, "", evSchema)

		} else if isActuator {
			var inputSchema *thing.DataSchema
			// only add data schema if the action accepts parameters
			if actuatorInfo.DataType != vocab.WoTDataTypeNone {
				unit, _ := UnitNameVocab[attr.Unit]
				inputSchema = &thing.DataSchema{
					Type:      actuatorInfo.DataType,
					Unit:      unit,
					ReadOnly:  false,
					WriteOnly: false,
				}
			}
			tdoc.AddAction(attrID, actuatorInfo.VocabType, actuatorInfo.Title, "", inputSchema)
		} else if !propIgnore {
			propType := ""
			title := attrID
			dataType := ""
			if isProp {
				propType = propInfo.VocabType
				title = propInfo.Title
				dataType = propInfo.DataType
			}
			prop := tdoc.AddProperty(attrID, propType, title, dataType)
			unit := propInfo.Unit
			if attr.Unit != "" {
				unit, _ = UnitNameVocab[attr.Unit]
			}
			prop.Unit = unit
			// non-sensors are attributes. Writable attributes are configuration.
			prop.ReadOnly = !attr.Writable
			if isProp && propInfo.Enum != nil {
				prop.SetEnumValues(propInfo.Enum)
			}
		}
	}
	return
}

//func (svc *OWServerBinding) MakeNodePropValues(node *eds.OneWireNode) map[string]string {
//	pv := make(map[string]string)
//	// Map node attribute to Thing properties
//	for attrName, attr := range node.Attr {
//		pv[attrName] = attr.Value
//	}
//
//	return pv
//}

// PollNodes polls the OWServer gateway for nodes and property values
func (svc *OWServerBinding) PollNodes() ([]*eds.OneWireNode, error) {
	nodes, err := svc.edsAPI.PollNodes()
	for _, node := range nodes {
		svc.nodes[node.ROMId] = node
	}
	return nodes, err
}

// PublishNodeTDs converts the nodes to TD documents and publishes these on the Hub message bus.
// This returns an error if one or more publications fail
func (svc *OWServerBinding) PublishNodeTDs(nodes []*eds.OneWireNode) (err error) {
	for _, node := range nodes {
		td := CreateTDFromNode(node)
		err2 := svc.hc.PubTD(td)
		if err2 != nil {
			err = err2
		} else {
			//props := svc.MakeNodePropValues(node)
			//_ = svc.hc.PubProps(td.ID, props)
		}
	}
	return err
}
