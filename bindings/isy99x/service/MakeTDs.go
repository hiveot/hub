package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/isy99x/service/isyapi"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
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
	prop := td.AddProperty(vocab.VocabPollInterval, vocab.VocabPollInterval, "Poll Interval", vocab.WoTDataTypeInteger, "")
	prop.Unit = vocab.UnitNameSecond
	prop.InitialValue = fmt.Sprintf("%d %s", svc.config.PollInterval, vocab.UnitNameSecond)

	prop = td.AddProperty("address", vocab.VocabGatewayAddress, "OWServer gateway IP address", vocab.WoTDataTypeString, "")
	prop.InitialValue = fmt.Sprintf("%s", svc.config.IsyAddress)

	// gateway events
	_ = td.AddEvent(vocab.VocabConnected, vocab.VocabConnected, "Connected to OWServer gateway", vocab.WoTDataTypeBool, nil)
	_ = td.AddEvent(vocab.VocabDisconnected, vocab.VocabDisconnected, "Connected lost to OWServer gateway", vocab.WoTDataTypeBool, nil)

	// no gateway actions

	return td
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
	td.AddPropertyAsString(vocab.VocabManufacturer, vocab.VocabManufacturer,
		"Manufacturer", isyDevice.Configuration.DeviceSpecs.Make) // Universal Devices Inc.
	td.AddPropertyAsString(vocab.VocabModel, vocab.VocabModel,
		"Model", isyDevice.Configuration.Platform) // ISY-C-99
	td.AddPropertyAsString(vocab.VocabSoftwareVersion, vocab.VocabSoftwareVersion,
		"AppVersion", isyDevice.Configuration.AppVersion) // 3.2.6
	td.AddPropertyAsString(vocab.VocabMAC, vocab.VocabMAC,
		"MAC", isyDevice.Configuration.Root.ID) // 00:21:xx:yy:... (mac)
	td.AddPropertyAsString(vocab.VocabProduct, vocab.VocabProduct,
		"Product", isyDevice.Configuration.Product.Description) // ISY 99i 256
	td.AddPropertyAsString("ProductID", vocab.VocabProduct,
		"Product ID", isyDevice.Configuration.Product.ID) // 1020
	prop := td.AddPropertyAsInt("sunrise", "",
		"Sunrise in seconds since epoch", isyDevice.Time.Sunrise)
	prop.Unit = vocab.UnitNameSecond
	prop = td.AddPropertyAsInt("sunset", "",
		"Sunset in seconds since epoch", isyDevice.Time.Sunrise)
	prop.Unit = vocab.UnitNameSecond

	// device configuration
	prop = td.AddPropertyAsString(vocab.VocabName, vocab.VocabName,
		"Name", isyDevice.Configuration.Root.Name) // custom name
	prop.ReadOnly = false

	// network config
	prop = td.AddPropertyAsBool("DHCP", "",
		"DHCP enabled", isyDevice.Network.Interface.IsDHCP) //
	prop.ReadOnly = false
	prop = td.AddPropertyAsString(vocab.VocabLocalIP, vocab.VocabLocalIP,
		"IP address", isyDevice.Address) //
	prop.ReadOnly = isyDevice.Network.Interface.IsDHCP == false
	prop = td.AddPropertyAsString("Gateway login name", "",
		"Name", svc.config.LoginName)
	prop.ReadOnly = false
	prop = td.AddPropertyAsString("Gateway login password", "",
		"Password", "")
	prop.WriteOnly = true

	// time config
	prop = td.AddPropertyAsString("NTPHost", "",
		"Network Time Host", isyDevice.System.NTPHost)
	prop.ReadOnly = false
	prop = td.AddPropertyAsBool("NTPEnabled", "",
		"NTP Enabled", isyDevice.System.NTPEnabled)
	prop.ReadOnly = false
	td.AddPropertyAsBool("DSTEnabled", "",
		"DST Enabled", isyDevice.Time.DST)
	prop.ReadOnly = false

	// TODO: any events?
	// TODO: any actions?
	//
	return td
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
	prop := td.AddPropertyAsInt(
		"flag", "", "Node Flag", int(node.Flag))
	prop.Description = "A bit mask: 0x01 -- Node is initialized (internal)," +
		" 0x02 -- Node is going to be crawled (internal)," +
		" 0x04 -- This is a group node," +
		" 0x08 -- This is the root node for ISY, i.e. My Lighting," +
		" 0x10 -- Device Communications Error," +
		" 0x20 -- Brand new node," +
		" 0x40 -- Node shall be deleted," +
		" 0x80 -- Node is device root"

	prop = td.AddPropertyAsString(
		"nodeType", "", "Node Type", node.Type)
	prop.Description = "<device cat>.<sub cat>.<model>.<reserved>"

	prop = td.AddPropertyAsString(
		"enabled", "", "Is the node plugged in", node.Enabled)
	prop.Description = "Whether or not the node is enabled (plugged in). Note: this feature only works on 99 Series"

	prop = td.AddPropertyAsString(
		"property", "", "raw property field",
		fmt.Sprintf("property id='%s' value='%s' formatted='%s' uom='%s'",
			node.Property.ID, node.Property.Value, node.Property.Formatted, node.Property.UOM))
	prop.Description = "Device's property for troubleshooting."

	//--- Node config ---
	prop = td.AddPropertyAsString(
		vocab.VocabName, vocab.VocabName, "Name", node.Name)
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
