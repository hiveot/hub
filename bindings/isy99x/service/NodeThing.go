package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"sync"
)

// mapping from insteon device category to TD device type
var deviceCatMap = map[string]string{

	"0x00": "Reserved",                    //• 0x00 - Reserved
	"0x01": vocab.DeviceTypeDimmer,        //• 0x01 - Dimmer device
	"0x02": vocab.DeviceTypeOnOffSwitch,   //• 0x02 - Relay or on/off switch device
	"0x03": vocab.DeviceTypeNetwork,       //• 0x03 - Network device
	"0x04": vocab.DeviceTypeIrrigationCtl, //• 0x04 - Irrigation device
	"0x05": vocab.DeviceTypeClimateCtl,    //• 0x05 - Climate control device
	"0x06": vocab.DeviceTypePoolCtl,       //• 0x06 - Pool control device
	"0x07": vocab.DeviceTypeSensor,        //• 0x07 - Sensor or actuator device
	"0x08": "Home Entertainment Unit",
	"0x09": "Energy management",
}

// NodeThing is the generic base of Things constructed out of ISY Insteon nodes.
// Intended for building specialized Things or for defining a basic Thing.
// This implements the IThing interface.
type NodeThing struct {
	// The node ID, also used as the ThingID
	id string

	// device type derived from productInfo
	deviceType string

	// ISY device info
	productInfo InsteonProduct

	// handler that applies an action request. Set by the specialized Thing
	actionHandler func(tv *things.ThingValue) error

	// handler that applies a config request. Set by the specialized Thing
	configHandler func(tv *things.ThingValue) error

	// eventCB to invoke on Thing events, if set.
	eventCB func(eventID string, value string)

	// propCB to invoke on property value changes, if set.
	propCB func(propID string, value string)

	propValues *things.PropertyValues

	// protect access to property values
	mux sync.RWMutex

	// REST/SOAP/WS connection to the ISY hub
	ic *IsyConnection
}

// GetID return the Node's ID
// The NodeID is used as-is as the ThingID.
func (it *NodeThing) GetID() string {
	return it.id
}

// GetProps returns the attr and config property values
func (it *NodeThing) GetProps(onlyChanges bool) map[string]string {
	return it.propValues.GetValues(onlyChanges)
}

// GetTD return a basic TD document that describes the Thing represented here.
// The parent should add properties, events and actions specific to their capabilities.
func (it *NodeThing) GetTD() *things.TD {
	td := things.NewTD(it.id, it.productInfo.ProductName, it.deviceType)

	// Node read-only properties
	prop := td.AddPropertyAsInt("flag", "", "Node Flag")
	prop.Description = "A bit mask: 0x01 -- Node is initialized (internal)," +
		" 0x02 -- Node is going to be crawled (internal)," +
		" 0x04 -- This is a group node," +
		" 0x08 -- This is the root node for ISY, i.e. My Lighting," +
		" 0x10 -- Device Communications Error," +
		" 0x20 -- Brand new node," +
		" 0x40 -- Node shall be deleted," +
		" 0x80 -- Node is device root"

	prop = td.AddPropertyAsString("nodeType", "", "Insteon Device Type")
	prop.Description = "<device cat>.<sub cat>.<version>.<reserved>"
	prop = td.AddPropertyAsString(vocab.VocabProduct, vocab.VocabProduct, "Product Name")
	prop = td.AddPropertyAsString(vocab.VocabModel, vocab.VocabModel, "Product Model")
	prop = td.AddPropertyAsString(vocab.VocabHardwareVersion, vocab.VocabHardwareVersion, "Device version")

	prop = td.AddPropertyAsString("enabled", "", "Is the node plugged in")
	prop.Description = "Whether or not the node is enabled (plugged in). Note: this feature only works on 99 Series"
	prop.Enum = []interface{}{"enabled", "disabled"}

	//prop = td.AddPropertyAsString("property", "", "raw property field")
	//prop.Description = "Device's property for troubleshooting."

	//--- Node config ---
	prop = td.AddPropertyAsString(vocab.VocabName, vocab.VocabName, "Name")
	prop.ReadOnly = false
	return td
}

// GetValue returns the default 'value' property
func (it *NodeThing) GetValue() (string, bool) {
	return it.propValues.GetValue(vocab.VocabValue)
}

// HandleActionRequest invokes the action handler of the specialized thing
func (it *NodeThing) HandleActionRequest(tv *things.ThingValue) (err error) {
	if it.actionHandler == nil {
		err = fmt.Errorf("HandleActionRequest not supported for this thing")
	} else {
		err = it.actionHandler(tv)
	}
	return err
}

// HandleConfigRequest invokes the config handler of the specialized thing
func (it *NodeThing) HandleConfigRequest(tv *things.ThingValue) (err error) {
	if it.configHandler == nil {
		err = fmt.Errorf("HandleConfigRequest not supported for this thing")
	} else {
		err = it.configHandler(tv)
	}
	return err
}

// HandleConfig is an empty method that returns an error
//func (it *NodeThing) HandleConfig(propID string, data []byte) error {
//	err := fmt.Errorf("HandleConfig not supported for this thing")
//	return err
//}

// HandleValueUpdate provides an update of the Thing's value.
// Invoked by the gateway when polling node values.
// This submits an event to the registered callback if the value differs.
func (it *NodeThing) HandleValueUpdate(propID string, uom string, newValue string) error {
	it.mux.Lock()
	defer it.mux.Unlock()
	// TODO: translate the propID to hiveot vocabulary
	// TODO: include use of the uom
	changed := it.propValues.SetValue(propID, newValue)
	if changed && it.propCB != nil {
		it.propCB(propID, newValue)
	}
	return nil
}

// Init initializes the NodeThing base class
// This determines the device type from prodInfo and sets property values for
// product and model.
func (it *NodeThing) Init(ic *IsyConnection, node *IsyNode, prodInfo InsteonProduct, hwVersion string) {
	var found bool
	it.deviceType, found = deviceCatMap[prodInfo.Cat]
	if !found {
		it.deviceType = vocab.DeviceTypeUnknown
	}

	it.ic = ic
	it.id = node.Address
	it.productInfo = prodInfo
	it.propValues = things.NewPropertyValues()
	pv := it.propValues
	pv.SetValue(vocab.VocabDeviceType, it.deviceType)
	pv.SetValue(vocab.VocabProduct, prodInfo.ProductName)
	pv.SetValue(vocab.VocabModel, prodInfo.Model)
	pv.SetValue(vocab.VocabHardwareVersion, hwVersion)
	pv.SetValue("nodeType", node.Type)
}

// Rename sets a new friendly name of the ISY device
func (it *NodeThing) Rename(newName string) error {
	// Post a SOAP message as no REST api exists for this
	soapTemplate := `<s:Envelope><s:Body>
  <u:RenameNode xmlns:u="urn:udi-com:service:X_Insteon_Lighting_Service:1">
    <id>%s</id>
    <name>%s</name>
  </u:RenameNode>
</s:Body></s:Envelope>
`
	msgBody := fmt.Sprint(soapTemplate, it.id, newName)
	err := it.ic.SendRequest("GET", msgBody, nil)
	if err != nil {
		// TODO: wait for update until event from gateway, needs WS support.
		it.propValues.SetValue(vocab.VocabName, newName)
		it.propCB(vocab.VocabName, newName)
	}
	return err
}

// SetEventCB sets the callback to invoke on ISY device events
func (it *NodeThing) SetEventCB(cb func(eventID string, value string)) {
	it.eventCB = cb
}

// SetPropCB sets the callback to invoke on ISY device property changes
func (it *NodeThing) SetPropCB(cb func(propID string, value string)) {
	it.propCB = cb
}

// NewIsyThing create a new instance of the generic ISY Insteon device Thing
// Call Init before use.
func NewIsyThing() *NodeThing {
	it := &NodeThing{}
	return it
}
