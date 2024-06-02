package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"strings"
	"sync"
)

// mapping from insteon device category to TD device type
var deviceCatMap = map[string]string{

	"0x00": "Reserved",                   //• 0x00 - Reserved
	"0x01": vocab.ThingActuatorDimmer,    //• 0x01 - Dimmer device
	"0x02": vocab.ThingActuatorSwitch,    //• 0x02 - Relay or on/off switch device
	"0x03": vocab.ThingNet,               //• 0x03 - Network device
	"0x04": vocab.ThingControlIrrigation, //• 0x04 - Irrigation device
	"0x05": vocab.ThingControlClimate,    //• 0x05 - Climate control device
	"0x06": vocab.ThingControlPool,       //• 0x06 - Pool control device
	"0x07": vocab.ThingActuator,          //• 0x07 - Sensor or actuator device
	"0x08": "Home Entertainment Unit",
	"0x09": "Energy management",
}

// IIsyThing is the interface implemented by nodes that are things
type IIsyThing interface {
	// GetID returns the thingID of the node
	GetID() string
	// GetValues returns the property values of the thing
	GetPropValues(onlyChanges bool) map[string]string
	// GetTD returns the generated TD document describing the Thing
	GetTD() *things.TD
	// HandleActionRequest passes incoming actions to the Thing for execution
	HandleActionRequest(tv *things.ThingMessage) (err error)
	// HandleConfigRequest passes configuration changes to the Thing for execution
	HandleConfigRequest(tv *things.ThingMessage) (err error)
	// HandleValueUpdate updates the Thing properties with value obtained via the ISY gateway
	HandleValueUpdate(propID string, uom string, newValue string) error
	// Init assigns the ISY connection and node this Thing represents
	Init(ic *IsyAPI, thingID string, node *IsyNode, prodInfo InsteonProduct, hwVersion string)
}

// IsyThing is the generic base of Things constructed out of ISY Insteon nodes.
// Intended for building specialized Things or for defining a basic Thing.
// This implements the IThing interface.
type IsyThing struct {
	// The node ID, also used as the ThingID
	nodeID string

	// ThingID derived from the nodeID
	thingID string

	// device type derived from productInfo
	deviceType string

	// ISY device info
	productInfo InsteonProduct

	// propValues holds the values of the thing properties
	propValues *things.PropertyValues

	// protect access to property values
	mux sync.RWMutex

	// REST/SOAP/WS connection to the ISY hub
	isyAPI *IsyAPI
}

// GetID returns the ThingID for the node it represents.
// No assumptions should be made on how this is constructed. The only
// guarantee is that it identifies, directly or indirectly, the node.
func (it *IsyThing) GetID() string {
	return it.thingID
}

// GetPropValues returns the property values
func (it *IsyThing) GetPropValues(onlyChanges bool) map[string]string {
	propValues := it.propValues.GetValues(onlyChanges)
	return propValues
}

// GetTD return a basic TD document that describes the Thing represented here.
// The parent should add properties, events and actions specific to their capabilities.
func (it *IsyThing) GetTD() *things.TD {
	title, _ := it.propValues.GetValue(vocab.PropDeviceTitle)
	if title == "" {
		title = it.productInfo.ProductName
	}
	it.mux.RLock()
	td := things.NewTD(it.thingID, title, it.deviceType)
	it.mux.RUnlock()

	//--- read-only properties
	prop := td.AddPropertyAsInt("flag", "", "Node Flag")
	prop.Description = "A bit mask: 0x01 -- Node is initialized (internal)," +
		" 0x02 -- Node is going to be crawled (internal)," +
		" 0x04 -- This is a group node," +
		" 0x08 -- This is the root node for ISY, i.e. My Lighting," +
		" 0x10 -- Device Communications Error," +
		" 0x20 -- Brand new node," +
		" 0x40 -- Node shall be deleted," +
		" 0x80 -- Node is device root"
	prop = td.AddPropertyAsString("nodeType", "", "Insteon device type")
	prop.Description = "<device cat>.<sub cat>.<version>.<reserved>"
	prop = td.AddPropertyAsString("", vocab.PropDeviceDescription, "Product description")
	prop = td.AddPropertyAsString("", vocab.PropDeviceModel, "Product model")
	prop = td.AddPropertyAsString("", vocab.PropDeviceHardwareVersion, "Device version")

	//--- configuration
	prop = td.AddPropertyAsString(vocab.PropDeviceEnabledDisabled,
		vocab.PropDeviceEnabledDisabled, "Enabled/disabled")
	prop.Description = "Whether or not the node is enabled (plugged in). Note: this feature only works on 99 Series"
	prop.Enum = []interface{}{"enabled", "disabled"}
	//prop.ReadOnly = false // TODO: support for enabled/disabled

	prop = td.AddPropertyAsString("", vocab.PropDeviceTitle, "Title")
	prop.ReadOnly = false
	return td
}

// GetValue returns the default 'value' property
//func (it *IsyThing) GetValue() (string, bool) {
//	return it.propValues.GetValue(vocab.PropDeviceValue)
//}

// HandleActionRequest invokes the action handler of the specialized thing
func (it *IsyThing) HandleActionRequest(tv *things.ThingMessage) (err error) {
	err = fmt.Errorf("HandleActionRequest not supported for this thing")
	return err
}

// HandleConfigRequest invokes the config handler of the specialized thing
func (it *IsyThing) HandleConfigRequest(action *things.ThingMessage) (err error) {
	// The title is the friendly name of the node
	if action.Key == vocab.PropDeviceTitle {
		newName := action.DataAsText()
		err = it.isyAPI.Rename(it.nodeID, newName)
		if err == nil {
			// TODO: use WebSocket to receive confirmation of change
			_ = it.HandleValueUpdate(vocab.PropDeviceTitle, "", newName)
		}
		return err
	}
	err = fmt.Errorf("HandleConfigRequest not supported for this thing")
	return err
}

// HandleValueUpdate provides an update of the Thing's value.
// Invoked by the gateway thing when polling node values.
// This submits an event to the registered callback if the value differs.
func (it *IsyThing) HandleValueUpdate(propID string, uom string, newValue string) (err error) {
	it.mux.Lock()
	it.propValues.SetValue(propID, newValue)
	it.mux.Unlock()

	return err
}

// Init initializes the IsyThing base class
// This determines the device type from prodInfo and sets property values for
// product and model.
func (it *IsyThing) Init(ic *IsyAPI, thingID string, node *IsyNode, prodInfo InsteonProduct, hwVersion string) {
	var found bool
	it.mux.Lock()
	defer it.mux.Unlock()
	it.deviceType, found = deviceCatMap[prodInfo.Cat]
	if !found {
		it.deviceType = vocab.ThingDevice
	}

	it.isyAPI = ic
	it.nodeID = node.Address
	it.thingID = thingID
	it.productInfo = prodInfo
	it.propValues = things.NewPropertyValues()
	enabledDisabled := "enabled"
	if strings.ToLower(node.Enabled) != "true" {
		enabledDisabled = "disabled"
	}
	pv := it.propValues
	pv.SetValue("deviceType", it.deviceType)
	pv.SetValue("flag", fmt.Sprintf("0x%X", node.Flag))
	pv.SetValue(vocab.PropDeviceEnabledDisabled, enabledDisabled)
	pv.SetValue(vocab.PropDeviceDescription, prodInfo.ProductName)
	pv.SetValue(vocab.PropDeviceTitle, node.Name)
	pv.SetValue(vocab.PropDeviceModel, prodInfo.Model)
	pv.SetValue(vocab.PropDeviceHardwareVersion, hwVersion)
	pv.SetValue("nodeType", node.Type)
}

// NewIsyThing constructs a general purpose ISY thing with basic properties
// Intended to be used for 'unknown' nodes.
// Init() must be called before use.
func NewIsyThing() *IsyThing {
	it := &IsyThing{}
	return it
}
