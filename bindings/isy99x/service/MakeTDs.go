package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/isy99x/service/isyapi"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"strconv"
	"strings"
)

// mapping from insteon device category to to TD device type
var deviceCatMap = map[string]string{
	"00": "Reserved",                    //• 0x00 - Reserved
	"01": vocab.DeviceTypeDimmer,        //• 0x01 - Dimmer device
	"02": vocab.DeviceTypeOnOffSwitch,   //• 0x02 - Relay or on/off switch device
	"03": vocab.DeviceTypeNetwork,       //• 0x03 - Network device
	"04": vocab.DeviceTypeIrrigationCtl, //• 0x04 - Irrigation device
	"05": vocab.DeviceTypeClimateCtl,    //• 0x05 - Climate control device
	"06": vocab.DeviceTypePoolCtl,       //• 0x06 - Pool control device
	"07": vocab.DeviceTypeSensor,        //• 0x07 - Sensor device
}

// map of ISY property names to hiveot vocabulary
var propNameMap = map[string]string{
	"ST": vocab.VocabOnOffSwitch, // relay/switch status
	"OL": vocab.VocabDimmer,      // on level
	// FIXME: these are control values, not property names
	"CLIFS":  vocab.VocabOnOffSwitch, // fan state On/Auto
	"CLIHUM": vocab.VocabSensor,      // humidity
	"CLIHC":  vocab.VocabOnOffSwitch, // climate heat/cool state
}

// getDeviceType determines the vocabulary name of the node
// if the property ID is unknown, itself is returned
func (svc *IsyBinding) getNodeTypeName(node *isyapi.IsyNode) (name string) {
	// FIXME: is there a better way to determine the device name?
	// determine what device this is
	parts := strings.Split(node.Type, ".")
	if len(parts) >= 3 {
		var found bool
		deviceCat := parts[0]
		//deviceSubcat := parts[1]
		//deviceModel := parts[2]
		name, found = deviceCatMap[deviceCat]
		if !found {
			name = vocab.DeviceTypeUnknown
		}
	}
	return name
}

// getPropName determines the vocabulary name of the ISY property
// if the property ID is unknown, itself is returned
func (svc *IsyBinding) getPropName(prop *isyapi.IsyProp) string {
	// FIXME: find a better way to map properties to their name
	// prop.ID refers to the control definition in config.
	// how to determine if a property is a temperature, humidity or other sensor?
	name, found := propNameMap[prop.ID]
	if !found {
		name = prop.ID
	}
	return name
}

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IsyBinding) MakeBindingTD() *things.TD {
	thingID := svc.hc.ClientID()
	td := things.NewTD(thingID, "OWServer binding", vocab.DeviceTypeBinding)

	// these are configured through the configuration file.
	prop := td.AddProperty(vocab.VocabPollInterval, vocab.VocabPollInterval, "Poll Interval", vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitNameSecond
	prop = td.AddProperty(vocab.VocabGatewayAddress, vocab.VocabGatewayAddress, "OWServer gateway IP address", vocab.WoTDataTypeString)

	// gateway events
	_ = td.AddEvent(vocab.VocabConnected, vocab.VocabConnected, "Connected to OWServer gateway", vocab.WoTDataTypeBool, nil)
	_ = td.AddEvent(vocab.VocabDisconnected, vocab.VocabDisconnected, "Connected lost to OWServer gateway", vocab.WoTDataTypeBool, nil)

	// no gateway actions
	return td
}

func (svc *IsyBinding) MakeBindingProps() map[string]string {
	pv := make(map[string]string)
	pv[vocab.VocabPollInterval] = fmt.Sprintf("%d", svc.config.PollInterval)
	pv[vocab.VocabGatewayAddress] = svc.config.IsyAddress
	return pv
}

// MakeGatewayTD creates a TD of the ISY gateway device
func (svc *IsyBinding) MakeGatewayTD(isyDevice *isyapi.IsyDevice) *things.TD {
	// update the TD
	// some fields to pick from

	// FIXME: support multiple instances
	thingID := isyDevice.Configuration.App             // Insteon_UD99
	title := isyDevice.Configuration.DeviceSpecs.Model // Insteon Web Controller
	deviceType := vocab.DeviceTypeGateway
	td := things.NewTD(thingID, title, deviceType)

	// device read-only attributes
	td.AddPropertyAsString(vocab.VocabManufacturer, vocab.VocabManufacturer, "Manufacturer")     // Universal Devices Inc.
	td.AddPropertyAsString(vocab.VocabModel, vocab.VocabModel, "Model")                          // ISY-C-99
	td.AddPropertyAsString(vocab.VocabSoftwareVersion, vocab.VocabSoftwareVersion, "AppVersion") // 3.2.6
	td.AddPropertyAsString(vocab.VocabMAC, vocab.VocabMAC, "MAC")                                // 00:21:xx:yy:... (mac)
	td.AddPropertyAsString(vocab.VocabProduct, vocab.VocabProduct, "Product")                    // ISY 99i 256
	td.AddPropertyAsString("ProductID", vocab.VocabProduct, "Product ID")                        // 1020
	prop := td.AddPropertyAsInt("sunrise", "", "Sunrise in seconds since epoch")
	prop.Unit = vocab.UnitNameSecond
	prop = td.AddPropertyAsInt("sunset", "", "Sunset in seconds since epoch")
	prop.Unit = vocab.UnitNameSecond

	// device configuration
	// custom name
	prop = td.AddPropertyAsString(vocab.VocabName, vocab.VocabName, "Name")
	prop.ReadOnly = false

	// network config
	prop = td.AddPropertyAsBool("DHCP", "", "DHCP enabled")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString(vocab.VocabLocalIP, vocab.VocabLocalIP, "IP address")
	prop.ReadOnly = isyDevice.Network.Interface.IsDHCP == false
	prop = td.AddPropertyAsString("Gateway login name", "", "Name")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString("Gateway login password", "", "Password")
	prop.WriteOnly = true

	// time config
	prop = td.AddPropertyAsString("NTPHost", "", "Network Time Host")
	prop.ReadOnly = false
	prop = td.AddPropertyAsBool("NTPEnabled", "", "NTP Enabled")
	prop.ReadOnly = false
	td.AddPropertyAsBool("DSTEnabled", "", "DST Enabled")
	prop.ReadOnly = false

	// TODO: any events?
	// TODO: any actions?
	//
	return td
}

// MakeGatewayProps generates a properties map for attribute and config properties of ISY gateway device
func (svc *IsyBinding) MakeGatewayProps(isyDevice *isyapi.IsyDevice) map[string]string {
	pv := make(map[string]string)
	pv[vocab.VocabManufacturer] = isyDevice.Configuration.DeviceSpecs.Make
	pv[vocab.VocabModel] = isyDevice.Configuration.Platform
	pv[vocab.VocabSoftwareVersion] = isyDevice.Configuration.AppVersion
	pv[vocab.VocabMAC] = isyDevice.Configuration.Root.ID
	pv[vocab.VocabProduct] = isyDevice.Configuration.Product.Description
	pv["ProductID"] = isyDevice.Configuration.Product.ID
	pv["sunrise"] = fmt.Sprintf("%d", isyDevice.Time.Sunrise)           // seconds since epoc
	pv["sunset"] = fmt.Sprintf("%d", isyDevice.Time.Sunset)             // seconds since epoc
	pv[vocab.VocabName] = isyDevice.Configuration.Root.Name             // custom name
	pv["DHCP"] = strconv.FormatBool(isyDevice.Network.Interface.IsDHCP) // true or false
	pv[vocab.VocabLocalIP] = isyDevice.Address
	pv["Gateway login name"] = svc.config.LoginName
	pv["NTPHost"] = isyDevice.System.NTPHost
	pv["NTPEnabled"] = strconv.FormatBool(isyDevice.System.NTPEnabled)
	pv["DSTEnabled"] = strconv.FormatBool(isyDevice.Time.DST)
	return pv
}

// MakeNodeTD makes a TD document of an isy node
func (svc *IsyBinding) MakeNodeTD(node *isyapi.IsyNode) (td *things.TD) {
	deviceType := vocab.DeviceTypeUnknown

	// determine what device this is
	parts := strings.Split(node.Type, ".")
	if len(parts) >= 3 {
		var found bool
		deviceCat := parts[0]
		//deviceSubcat := parts[1]
		//deviceModel := parts[2]
		deviceType, found = deviceCatMap[deviceCat]
		if !found {
			deviceType = vocab.DeviceTypeUnknown
		}
	}
	td = things.NewTD(node.Address, node.Name, deviceType)

	//--- Node attributes (read-only) that describe the device ---
	prop := td.AddPropertyAsInt("flag", "", "Node Flag")
	prop.Description = "A bit mask: 0x01 -- Node is initialized (internal)," +
		" 0x02 -- Node is going to be crawled (internal)," +
		" 0x04 -- This is a group node," +
		" 0x08 -- This is the root node for ISY, i.e. My Lighting," +
		" 0x10 -- Device Communications Error," +
		" 0x20 -- Brand new node," +
		" 0x40 -- Node shall be deleted," +
		" 0x80 -- Node is device root"

	prop = td.AddPropertyAsString("nodeType", "", "Node Type")
	prop.Description = "<device cat>.<sub cat>.<model>.<reserved>"

	prop = td.AddPropertyAsString("enabled", "", "Is the node plugged in")
	prop.Description = "Whether or not the node is enabled (plugged in). Note: this feature only works on 99 Series"

	prop = td.AddPropertyAsString("property", "", "raw property field")
	prop.Description = "Device's property for troubleshooting."

	//--- Node config ---
	prop = td.AddPropertyAsString(vocab.VocabName, vocab.VocabName, "Name")
	prop.ReadOnly = false

	//--- Node events (outputs) depend on the device type ---
	propName := svc.getPropName(&node.Property)
	if deviceType == vocab.DeviceTypeOnOffSwitch {
		//v := node.Property.Value
		//initialValue := true
		//if v == "" || v == "DOF" || v == "0" || strings.ToLower(v) == "false" {
		//	initialValue = false
		//}
		td.AddSwitchEvent(propName)
	} else if deviceType == vocab.DeviceTypeDimmer {
		td.AddDimmerEvent(propName)
	} else {
		td.AddSensorEvent(propName)
	}

	//--- Node actions (inputs) depend on the device type ---
	// TODO: dimmer:set value; switch: onoff action
	if deviceType == vocab.DeviceTypeDimmer {
		a := td.AddDimmerAction(propName)
		a.Input.Unit = vocab.UnitNamePercent
		a.Input.NumberMinimum = 0
		a.Input.NumberMaximum = 100
	} else if deviceType == vocab.DeviceTypeOnOffSwitch {
		td.AddSwitchAction(propName)
	}
	return td
}

// MakeNodeProps generates a properties map for attribute and config properties of ISY gateway device
func (svc *IsyBinding) MakeNodeProps(node *isyapi.IsyNode) map[string]string {
	pv := make(map[string]string)
	pv["flag"] = fmt.Sprintf("%d", node.Flag)
	pv["nodeType"] = node.Type
	pv["enabled"] = node.Enabled
	pv["property"] = fmt.Sprintf("property id='%s' value='%s' formatted='%s' uom='%s'",
		node.Property.ID, node.Property.Value, node.Property.Formatted, node.Property.UOM)
	pv[vocab.VocabName] = node.Name
	return pv
}
