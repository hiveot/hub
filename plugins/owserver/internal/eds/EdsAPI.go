// Package eds with EDS OWServer API methods
package eds

import (
	"encoding/xml"
	"fmt"
	"golang.org/x/exp/slog"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hiveot/hub/api/go/vocab"
)

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
// Todo: get from config file so it is easy to update
var deviceTypeMap = map[string]string{
	"01": "serialNumber",               // 2401,2411 (1990A): Silicon Serial Number
	"02": "securityKey",                // 1425 (1991): multikey 1153bit secure
	"04": vocab.DeviceTypeTime,         // 2404 (1994): econoram time chip
	"05": vocab.DeviceTypeBinarySwitch, // 2405: Addresable Switch
	"06": vocab.DeviceTypeMemory,       // (1993) 4K memory ibutton
	"08": vocab.DeviceTypeMemory,       // (1992) 1K memory ibutton
	"0A": vocab.DeviceTypeMemory,       // (1995) 16K memory ibutton
	"0C": vocab.DeviceTypeMemory,       // (1996) 64K memory ibutton
	"10": vocab.DeviceTypeThermometer,  // 18S20: high precision digital thermometer
	"12": vocab.DeviceTypeBinarySwitch, // 2406:  dual addressable switch plus 1k memory
	"14": vocab.DeviceTypeEeprom,       // 2430A (1971): 256 EEPROM
	"1A": vocab.DeviceTypeEeprom,       // (1963L) 4K Monetary
	"1C": vocab.DeviceTypeEeprom,       // 28E04-100: 4K EEPROM with PIO
	"1D": vocab.DeviceTypeMemory,       // 2423:  4K RAM with counter
	"1F": "coupler",                    // 2409:  Microlan coupler?
	"20": "adconverter",                // 2450:  Quad A/D convert
	"21": vocab.DeviceTypeThermometer,  // 1921:  Thermochron iButton device
	"22": vocab.DeviceTypeThermometer,  // 1822:  Econo digital thermometer
	"24": vocab.DeviceTypeTime,         // 2415:  time chip
	"26": vocab.DeviceTypeBatteryMon,   // 2438:  smart battery monitor
	"27": vocab.DeviceTypeTime,         // 2417:  time chip with interrupt
	"28": vocab.DeviceTypeThermometer,  // 18B20: programmable resolution digital thermometer
	"29": vocab.DeviceTypeOnOffSwitch,  // 2408:  8-channel addressable switch
	"2C": vocab.DeviceTypeSensor,       // 2890:  digital potentiometer"
	"2D": vocab.DeviceTypeEeprom,       // 2431:  1k eeprom
	"2E": vocab.DeviceTypeBatteryMon,   // 2770:  battery monitor and charge controller
	"30": vocab.DeviceTypeBatteryMon,   // 2760, 2761, 2762:  high-precision li+ battery monitor
	"31": vocab.DeviceTypeBatteryMon,   // 2720: efficient addressable single-cell rechargable lithium protection ic
	"33": vocab.DeviceTypeEeprom,       // 2432 (1961S): 1k protected eeprom with SHA-1
	"36": vocab.DeviceTypeSensor,       // 2740: high precision coulomb counter
	"37": vocab.DeviceTypeEeprom,       // (1977): Password protected 32k eeprom
	"3B": vocab.DeviceTypeThermometer,  // DS1825: programmable digital thermometer (https://www.analog.com/media/en/technical-documentation/data-sheets/ds1825.pdf)
	"41": vocab.DeviceTypeSensor,       // 2422: Temperature Logger 8k mem
	"42": vocab.DeviceTypeThermometer,  // DS28EA00: digital thermometer with PIO (https://www.analog.com/media/en/technical-documentation/data-sheets/ds28ea00.pdf)
	"51": vocab.DeviceTypeIndicator,    // 2751: multi chemistry battery fuel gauge
	"84": vocab.DeviceTypeTime,         // 2404S: dual port plus time
	//# EDS0068: Temperature, Humidity, Barometric Pressure and Light Sensor
	//https://www.embeddeddatasystems.com/assets/images/supportFiles/manuals/EN-UserMan%20%20OW-ENV%20Sensor%20v13.pdf
	"7E": vocab.DeviceTypeMultisensor,
}

// AttrVocab maps OWServer attribute names to IoT vocabulary
var AttrVocab = map[string]string{
	"MACAddress": vocab.VocabMAC,
	//"DateTime":   vocab.VocabDateTime,
	"DeviceName": vocab.VocabName,
	"HostName":   vocab.VocabHostname,
	"Version":    vocab.VocabSoftwareVersion,
	// Exclude/ignore the following attributes as they are chatty or not useful
	"BarometricPressureHg":                           "",
	"BarometricPressureHgHighAlarmState":             "",
	"BarometricPressureHgHighAlarmValue":             "",
	"BarometricPressureHgHighConditionalSearchState": "",
	"BarometricPressureHgLowAlarmState":              "",
	"BarometricPressureHgLowAlarmValue":              "",
	"BarometricPressureHgLowConditionalSearchState":  "",
	"BarometricPressureMbHighConditionalSearchState": "",
	"BarometricPressureMbLowConditionalSearchState":  "",
	"Counter1":                              "",
	"Counter2":                              "",
	"DateTime":                              "",
	"DewPointHighConditionalSearchState":    "",
	"DewPointLowConditionalSearchState":     "",
	"HeatIndexHighConditionalSearchState":   "",
	"HeatIndexLowConditionalSearchState":    "",
	"Humidex":                               "",
	"HumidexHighAlarmState":                 "",
	"HumidexHighConditionalSearchState":     "",
	"HumidexLowAlarmState":                  "",
	"HumidexLowConditionalSearchState":      "",
	"HumidityHighConditionalSearchState":    "",
	"HumidityLowConditionalSearchState":     "",
	"LightHighConditionalSearchState":       "",
	"LightLowConditionalSearchState":        "",
	"TemperatureHighConditionalSearchState": "",
	"TemperatureLowConditionalSearchState":  "",
	"PollCount":                             "",
	"PrimaryValue":                          "",
	"RawData":                               "",
}

// SensorTypeVocab maps OWServer sensor names to IoT vocabulary
var SensorTypeVocab = map[string]struct {
	sensorType string // sensor type from vocabulary
	name       string
	dataType   string
	decimals   int // number of decimals accuracy for this value
}{
	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
	"BarometricPressureMb":               {sensorType: vocab.VocabAtmosphericPressure, name: "Atmospheric Pressure", dataType: vocab.WoTDataTypeNumber, decimals: 0}, // unit Mb
	"BarometricPressureMbHighAlarmState": {sensorType: vocab.VocabAlarmState, name: "Pressure High Alarm", dataType: vocab.WoTDataTypeBool},
	"BarometricPressureMbLowAlarmState":  {sensorType: vocab.VocabAlarmState, name: "Pressure Low Alarm", dataType: vocab.WoTDataTypeBool},
	"DewPoint":                           {sensorType: vocab.VocabDewpoint, name: "Dew point", dataType: vocab.WoTDataTypeNumber, decimals: 1},
	"Health":                             {sensorType: "health", name: "Health 0-7", dataType: vocab.WoTDataTypeNumber},
	"HeatIndex":                          {sensorType: vocab.VocabHeatIndex, name: "Heat Index", dataType: vocab.WoTDataTypeNumber, decimals: 1},
	"Humidity":                           {sensorType: vocab.VocabHumidity, name: "Humidity", dataType: vocab.WoTDataTypeNumber, decimals: 0},
	"HumidityHighAlarmState":             {sensorType: vocab.VocabAlarmState, name: "Humidity High Alarm", dataType: vocab.WoTDataTypeBool},
	"HumidityLowAlarmState":              {sensorType: vocab.VocabAlarmState, name: "Humidity Low Alarm", dataType: vocab.WoTDataTypeBool},
	"Light":                              {sensorType: vocab.VocabLuminance, name: "Luminance", dataType: vocab.WoTDataTypeNumber, decimals: 0},
	"RelayState":                         {sensorType: vocab.VocabRelay, name: "Relay State", dataType: vocab.WoTDataTypeBool, decimals: 0},
	"Temperature":                        {sensorType: vocab.VocabTemperature, name: "Temperature", dataType: vocab.WoTDataTypeNumber, decimals: 1},
	"TemperatureHighAlarmState":          {sensorType: vocab.VocabAlarmState, name: "Temperature High Alarm", dataType: vocab.WoTDataTypeBool},
	"TemperatureLowAlarmState":           {sensorType: vocab.VocabAlarmState, name: "Temperature Low Alarm", dataType: vocab.WoTDataTypeBool},
}

// ActuatorTypeVocab maps OWServer names to IoT vocabulary
var ActuatorTypeVocab = map[string]struct {
	ActuatorType string // sensor type from vocabulary
	Title        string
	DataType     string
}{
	// "BarometricPressureHg": vocab.PropNameAtmosphericPressure, // unit Hg
	"Relay": {ActuatorType: vocab.VocabRelay, Title: "Relay", DataType: vocab.WoTDataTypeBool},
}

// UnitNameVocab maps OWServer unit names to IoT vocabulary
var UnitNameVocab = map[string]string{
	"PercentRelativeHumidity": vocab.UnitNamePercent,
	"Millibars":               vocab.UnitNameMillibar,
	"Centigrade":              vocab.UnitNameCelcius,
	"Fahrenheit":              vocab.UnitNameFahrenheit,
	"InchesOfMercury":         vocab.UnitNameMercury,
	"Lux":                     vocab.UnitNameLux,
	"//":                      vocab.UnitNameCount,
	"Volt":                    vocab.UnitNameVolt,
}

// EdsAPI EDS device API properties and methods
type EdsAPI struct {
	address         string     // EDS (IP) address or filename (file://./path/to/name.xml)
	loginName       string     // Basic Auth login name
	password        string     // Basic Auth password
	discoTimeoutSec int        // EDS OWServer discovery timeout
	readMutex       sync.Mutex // prevent concurrent discovery
}

// XMLNode XML parsing node. Pure magic...
// --- https://stackoverflow.com/questions/30256729/how-to-traverse-through-xml-data-in-golang
type XMLNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []XMLNode  `xml:",any"`
	// Possible attributes for subnodes, depending on the property name
	Description string `xml:"Description,attr"`
	Writable    string `xml:"Writable,attr"`
	Units       string `xml:"Units,attr"`
}

// OneWireAttr with info on each node attribute, property, event or action
type OneWireAttr struct {
	ID         string // attribute raw instance ID
	Name       string // attribute Title for humans
	VocabType  string // attribute type from vocabulary, if any, eg 'temperature', ...
	Unit       string
	Writable   bool
	Value      string
	IsActuator bool
	IsSensor   bool   // sensors emit events on change
	DataType   string // vocab data type, "string", "number", "boolean", ""
}

// OneWireNode with info on each node
type OneWireNode struct {
	DeviceType string
	// ThingID     string
	NodeID      string // ROM ID
	Name        string
	Description string
	Attr        map[string]OneWireAttr // attribute by ID
}

// Apply the vocabulary to the name
// This returns the translated name from the vocabulary or the original name if not in the vocabulary
func applyVocabulary(name string, vocab map[string]string) (vocabName string, hasName bool) {
	vocabName, hasName = vocab[name]
	if !hasName {
		vocabName = name
	}
	return vocabName, hasName
}

// Discover any EDS OWServer ENet-2 on the local network for 3 seconds
// This uses a UDP Broadcast on port 30303 as stated in the manual
// If found, this sets the service address for further use
// Returns the address or an error if not found
func Discover(timeoutSec int) (addr string, err error) {
	slog.Info("Starting discovery")
	var addr2 *net.UDPAddr
	// listen
	conn, err := net.ListenPacket("udp4", ":30303")
	if err == nil {
		defer conn.Close()

		addr2, err = net.ResolveUDPAddr("udp4", "255.255.255.255:30303")
	}
	if err == nil {
		_, err = conn.WriteTo([]byte("D"), addr2)
	}
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	// receive 2 messages, first the broadcast, followed by the response, if there is one
	// wait 3 seconds before giving up
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeoutSec)))
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			slog.Info("Discovery ended without results")
			return "", err
		} else if n > 1 {
			switch rxAddr := remoteAddr.(type) {
			case *net.UDPAddr:
				addr = rxAddr.IP.String()
				slog.Info("Found OwServer", "addr", addr, "record", buf[:n])
				return "http://" + addr, nil
			}
		}
	}
}

// GetLastAddress returns the last used address of the gateway
// This is either the configured or the discovered address
//func (edsAPI *EdsAPI) GetLastAddress() string {
//	return edsAPI.address
//}

// ParseOneWireNodes parses the owserver xml data and returns a list of nodes,
// including the owserver gateway, and their parameters.
// This also converts sensor values to a proper decimals. Eg temperature isn't 4 digits but 1.
//
//	xmlNode is the node to parse, its attribute and possibly subnodes
//	latency to add to the root node (gateway device)
//	isRootNode is set for the first node, eg the gateway itself
func ParseOneWireNodes(
	xmlNode *XMLNode, latency time.Duration, isRootNode bool) []*OneWireNode {

	owNodeList := make([]*OneWireNode, 0)

	owNode := OneWireNode{
		// ID:          xmlNode.Attrs["ROMId"],
		Name:        xmlNode.XMLName.Local,
		Description: xmlNode.Description,
		Attr:        make(map[string]OneWireAttr),
		DeviceType:  vocab.DeviceTypeGateway,
	}
	owNodeList = append(owNodeList, &owNode)
	// todo: find a better place for this
	if isRootNode {
		owAttr := OneWireAttr{
			ID:       vocab.VocabLatency,
			Name:     vocab.VocabLatency,
			Value:    fmt.Sprintf("%.2f", latency.Seconds()),
			Unit:     "sec",
			DataType: vocab.WoTDataTypeNumber,
		}
		owNode.Attr[owAttr.Name] = owAttr
	}
	// parse attributes and round sensor values
	for _, node := range xmlNode.Nodes {
		// if the xmlnode has no subnodes then it is a parameter describing the current node
		if len(node.Nodes) == 0 {
			// standardize the naming of properties and property types
			writable := strings.ToLower(node.Writable) == "true"
			attrID := node.XMLName.Local
			title := attrID
			actuatorInfo, isActuator := ActuatorTypeVocab[attrID]
			sensorInfo, isSensor := SensorTypeVocab[attrID]
			vocabType := "" // standardized type, if known
			decimals := -1  // -1 means no conversion
			dataType := vocab.WoTDataTypeString

			if isActuator {
				// this is a known actuator type
				title = actuatorInfo.Title
				vocabType = actuatorInfo.ActuatorType
				dataType = actuatorInfo.DataType
			} else if isSensor {
				// this is a known sensor type
				title = sensorInfo.name
				vocabType = sensorInfo.sensorType
				decimals = sensorInfo.decimals
				dataType = sensorInfo.dataType
			} else {
				// this is an attribute, or configuration when writable
				vocabType, _ = applyVocabulary(attrID, AttrVocab)
			}
			// ignore values erased in the vocabulary
			if vocabType != "" {
				unit, _ := applyVocabulary(node.Units, UnitNameVocab)
				valueStr := string(node.Content)
				valueFloat, err := strconv.ParseFloat(valueStr, 32)
				// if it can be parsed then it is a number
				if err == nil && dataType != vocab.WoTDataTypeBool {
					// rounding of sensor values to decimals
					if decimals >= 0 {
						ratio := math.Pow(10, float64(decimals))
						valueFloat = math.Round(valueFloat*ratio) / ratio
						valueStr = strconv.FormatFloat(valueFloat, 'f', decimals, 32)
					}
					dataType = vocab.WoTDataTypeNumber
				}

				owAttr := OneWireAttr{
					ID:         attrID,
					Name:       title,
					VocabType:  vocabType,
					Value:      valueStr,
					Unit:       unit,
					IsSensor:   isSensor,
					IsActuator: isActuator,
					Writable:   writable,
					DataType:   dataType,
				}
				owNode.Attr[owAttr.ID] = owAttr
				// Family is used to determine device type, default is gateway
				if node.XMLName.Local == "Family" {
					deviceType := deviceTypeMap[owAttr.Value]
					if deviceType == "" {
						deviceType = vocab.DeviceTypeUnknown
					}
					owNode.DeviceType = deviceType
				} else if node.XMLName.Local == "ROMId" {
					// all subnodes use the ROMId as its ID
					owNode.NodeID = owAttr.Value
				} else if isRootNode && node.XMLName.Local == "DeviceName" {
					// The gateway itself uses the deviceName as its ID and name
					owNode.NodeID = owAttr.Value
					owNode.Name = owAttr.Value
					owNode.Description = "EDS OWServer Gateway"
				}
			}
		} else {
			// The node contains subnodes which contain one or more sensors.
			subNodes := ParseOneWireNodes(&node, 0, false)
			owNodeList = append(owNodeList, subNodes...)
		}
	}
	// owNode.ThingID = td.CreatePublisherThingID(pb.hubConfig.Zone, PluginID, owNode.NodeID, owNode.DeviceType)

	return owNodeList
}

// PollNodes polls the OWServer gateway for nodes and property values
// Returns a list of nodes and a map of device/node ID's containing a map of property name:value
// pairs.
func (edsAPI *EdsAPI) PollNodes() (nodeList []*OneWireNode, err error) {

	// Read the values from the EDS gateway
	if edsAPI.address == "" {
		edsAPI.address, err = Discover(edsAPI.discoTimeoutSec)
		if err != nil {
			return nil, err
		}
	}
	startTime := time.Now()
	rootNode, err := ReadEds(edsAPI.address, edsAPI.loginName, edsAPI.password)
	endTime := time.Now()
	latency := endTime.Sub(startTime)
	if err != nil {
		slog.Error("failed", "err", err)
		return nil, err
	}

	// Extract the nodes and convert properties to vocab names
	nodeList = ParseOneWireNodes(rootNode, latency, true)
	slog.Info("Nodes found", "count", len(nodeList))
	return nodeList, nil
}

// ReadEds reads EDS gateway and return the result as an XML node
// If edsAPI.address starts with file:// then read from file, otherwise from http
// If no address is configured, one will be auto discovered the first time.
func ReadEds(address, loginName, password string) (rootNode *XMLNode, err error) {
	// don't discover or read concurrently
	if strings.HasPrefix(address, "file://") {
		filename := address[7:]
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Unable to read EDS file", "err", err, "filename", filename)
			return nil, err
		}
		err = xml.Unmarshal(buffer, &rootNode)
		return rootNode, err
	}
	// not a file, continue with http request
	edsURL := address + "/details.xml"
	req, _ := http.NewRequest("GET", edsURL, nil)

	req.SetBasicAuth(loginName, password)
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)

	// resp, err := http.Get(edsURL)
	if err != nil {
		slog.Error("Unable to read EDS gateway", "err", err.Error(), "url", edsURL)
		return nil, err
	}
	// Decode the EDS response into XML
	dec := xml.NewDecoder(resp.Body)
	err = dec.Decode(&rootNode)
	_ = resp.Body.Close()

	return rootNode, err
}

// UnmarshalXML parse xml
func (n *XMLNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node XMLNode

	return d.DecodeElement((*node)(n), &start)
}

// WriteData writes a value to a variable
// this posts a request to devices.html?rom={romID}&variable={variable}&value={value}
func (edsAPI *EdsAPI) WriteData(romID string, variable string, value string) error {
	// TODO: auto config if this is http or https
	writeURL := edsAPI.address + "/devices.htm" +
		"?rom=" + romID + "&variable=" + variable + "&value=" + value
	req, _ := http.NewRequest("GET", writeURL, nil)

	slog.Info("write to OwServer", "URL", writeURL)
	req.SetBasicAuth(edsAPI.loginName, edsAPI.password)
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	_ = resp

	if err != nil {
		slog.Error("Unable to write data to EDS gateway", "err", err.Error(), "url", writeURL)
	}
	return err
}

// NewEdsAPI creates a new NewEdsAPI instance
//
//	address is optional to override the discovery
//	loginName if needed, "" if not needed
//	password if needed, "" if not needed
func NewEdsAPI(address string, loginName string, password string) *EdsAPI {
	edsAPI := &EdsAPI{
		address:         address,
		loginName:       loginName,
		password:        password,
		discoTimeoutSec: 3, // discovery timeout
	}
	return edsAPI
}
