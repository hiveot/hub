package internal

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hub/bindings/isy99x/internal/isy"
)

// mapping from insteon device category to TD device type
var deviceCatMap = map[string]string{

	"0x00": "Reserved",                    //• 0x00 - Reserved
	"0x01": vocab.DeviceActuatorDimmer,    //• 0x01 - Dimmer device
	"0x02": vocab.DeviceActuatorSwitch,    //• 0x02 - Relay or on/off switch device
	"0x03": vocab.DeviceNet,               //• 0x03 - Network device
	"0x04": vocab.DeviceControlIrrigation, //• 0x04 - Irrigation device
	"0x05": vocab.DeviceControlClimate,    //• 0x05 - Climate control device
	"0x06": vocab.DeviceControlPool,       //• 0x06 - Pool control device
	"0x07": vocab.DeviceActuator,          //• 0x07 - Sensor or actuator device
	"0x08": "Home Entertainment Unit",
	"0x09": "Energy management",
}

// IsyEventHandler is the callback to notify the binding of thing events
type IsyEventHandler func(thingID, name string, value any)

// IIsyThing is the interface implemented by nodes that are things
type IIsyThing interface {
	// *thing.ExposedThing
	api.IHiveCell

	// GetID returns the thingID of the node
	// GetID() string

	// RefreshProperties loads and publishes node property values
	RefreshProperties(onlyChanges bool)

	// HandleActionRequest passes incoming actions to the Thing for execution
	// HandleActionRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error
	// HandleConfigRequest passes configuration changes to the Thing for execution
	// HandleConfigRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// HandleValueUpdate updates the Thing properties with value obtained via the ISY gateway
	HandleValueUpdate(propID string, uom string, newValue string) error

	// Init assigns the ISY connection and node this Thing represents
	// Init(ic *isy.IsyAPI, thingID string, node *isy.IsyNode, prodInfo InsteonProduct, hwVersion string)
	// MakeTD returns the generated TD document describing the Thing
	// MakeTD() *td.TD
}

// IsyThing is the generic base of Things constructed out of ISY Insteon nodes.
// Intended for building specialized Things or for defining a basic Thing.
// This implements the IThing interface.
type IsyThing struct {
	*thing.ExposedThing

	// The underlying ISY node
	node *isy.IsyNode

	// vocab device type derived from productInfo (@type)
	deviceType string

	// ISY device info
	productInfo InsteonProduct

	// protect access to property values
	mux sync.RWMutex

	// REST/SOAP/WS connection to the ISY hub
	isyAPI *isy.IsyGWAPI

	// The TD to publish on start
	tdoc *td.TD
}

// RefreshProperties returns the property values, set with read
func (it *IsyThing) RefreshProperties(onlyChanges bool) {
	// nothing to do here
}

// GetOutputValue returns the default 'value' property
//func (it *IsyThing) GetOutputValue() (string, bool) {
//	return it.propValues.GetOutputValue(vocab.PropDeviceValue)
//}

func (it *IsyThing) GetNode() *isy.IsyNode {
	return it.node
}

// HandleActionRequest invokes the action handler of the specialized thing
func (it *IsyThing) HandleActionRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	err := fmt.Errorf("HandleActionRequest this thing has no actions")
	return err
}

// HandleConfigRequest invokes the config handler of the specialized thing
func (it *IsyThing) HandleConfigRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	// The title is the friendly name of the node
	if req.Name == td.WoTTitle {
		newName := req.ToString(0)
		err := it.isyAPI.Rename(it.node.Address, newName)
		if err == nil {
			// TODO: use ISY WebSocket to receive confirmation of change
			_ = it.HandleValueUpdate(td.WoTTitle, "", newName)
		}
		resp := req.CreateResponse(nil, err)
		return replyTo(resp)
	}
	err := fmt.Errorf("HandleConfigRequest not supported for this thing")
	return err
}

// HandleValueUpdate provides an update of the Thing's value.
// Invoked by the gateway thing when polling node values.
// This submits an event to the registered callback if the value differs.
func (it *IsyThing) HandleValueUpdate(propID string, uom string, newValue string) (err error) {

	it.SetProperty(it.GetID(), propID, newValue)

	return err
}

// MakeTD return a basic TD document that describes the Thing represented here.
// The parent should add properties, events and actions specific to their capabilities.
func (it *IsyThing) MakeTD() *td.TD {
	title := it.productInfo.ProductName
	titleProp, _ := it.GetProperty(td.WoTTitle)
	if titleProp != nil {
		title, _ = titleProp.(string)
	}
	it.mux.RLock()
	tdoc := td.NewTD(it.GetID(), title, it.deviceType)
	it.mux.RUnlock()

	//--- read-only properties
	prop := tdoc.AddPropertyAsInt("flag", "Node Flag", "")
	prop.Description = "A bit mask: 0x01 -- Node is initialized (internal)," +
		" 0x02 -- Node is going to be crawled (internal)," +
		" 0x04 -- This is a group node," +
		" 0x08 -- This is the root node for ISY, i.e. My Lighting," +
		" 0x10 -- Device Communications Error," +
		" 0x20 -- Brand new node," +
		" 0x40 -- Node shall be deleted," +
		" 0x80 -- Node is device root"
	prop = tdoc.AddPropertyAsString("deviceType", "Hiveot device type", "")
	prop = tdoc.AddPropertyAsString("nodeType", "Insteon device type", "")
	prop.Description = "<device cat>.<sub cat>.<version>.<reserved>"
	prop = tdoc.AddPropertyAsString(vocab.PropDeviceDescription, "Product description", "").
		SetAtType(vocab.PropDeviceDescription)
	prop = tdoc.AddPropertyAsString(vocab.PropDeviceModel, "Product model", "").
		SetAtType(vocab.PropDeviceModel)
	prop = tdoc.AddPropertyAsString(vocab.PropDeviceHardwareVersion, "Hardware version", "Device hardware version").
		SetAtType(vocab.PropDeviceHardwareVersion)

	//--- configuration
	prop = tdoc.AddPropertyAsString(vocab.PropDeviceEnabledDisabled, "Enabled/disabled", "")
	prop.SetAtType(vocab.PropDeviceEnabledDisabled)
	prop.Description = "Whether or not the node is enabled (plugged in). Note: this feature only works on 99 Series"
	prop.Enum = []interface{}{"enabled", "disabled"}
	//prop.ReadOnly = false // TODO: support for enabled/disabled

	prop = tdoc.AddPropertyAsString(tdoc.Title, "Title", "").
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = false
	return tdoc
}

// Start using the ISY Thing.
//
// This publishes the Thing TD and its static properties.
func (it *IsyThing) Start() {

	it.WriteTD(it.tdoc)

	enabledDisabled := "enabled"
	if strings.ToLower(it.node.Enabled) != "true" {
		enabledDisabled = "disabled"
	}
	prodInfo := it.productInfo
	props := map[string]any{
		"deviceType":                    it.deviceType,
		"flag":                          fmt.Sprintf("0x%X", it.node.Flag),
		vocab.PropDeviceEnabledDisabled: enabledDisabled,
		vocab.PropDeviceDescription:     prodInfo.ProductName,
		td.WoTTitle:                     it.node.Name,
		vocab.PropDeviceModel:           prodInfo.Model,
		vocab.PropDeviceHardwareVersion: prodInfo.HWVersion,
		"nodeType":                      it.node.Type,
	}
	it.PubProperties("", props, false)
}

// PubPropValues gets the thing properties and publish them
//func (svc *IsyThing) PubPropValues(hc hubclient.IAgent, onlyChanges bool) (err error) {
//	props := svc.GetPropValues(onlyChanges)
//	if len(props) > 0 {
//		err = hc.PubMultipleProperties(svc.thingID, props)
//	}
//	return err
//}

// NewIsyThing constructs a general purpose ISY thing with basic properties
// Intended to be used for 'unknown' nodes.
func NewIsyThing(ic *isy.IsyGWAPI,
	thingID string, node *isy.IsyNode, prodInfo InsteonProduct, hwVersion string) *IsyThing {

	it := &IsyThing{
		ExposedThing: thing.NewExposedThing(thingID, nil),
		isyAPI:       ic,
		node:         node,
		productInfo:  prodInfo,
	}
	it.tdoc = it.MakeTD()

	deviceType, found := deviceCatMap[prodInfo.Cat]
	if !found {
		deviceType = vocab.Device
	}
	it.deviceType = deviceType
	it.isyAPI = ic
	prodInfo.HWVersion = hwVersion

	// enabledDisabled := "enabled"
	// if strings.ToLower(node.Enabled) != "true" {
	// 	enabledDisabled = "disabled"
	// }
	// props := map[string]any{
	// 	"deviceType":                    it.deviceType,
	// 	"flag":                          fmt.Sprintf("0x%X", node.Flag),
	// 	vocab.PropDeviceEnabledDisabled: enabledDisabled,
	// 	vocab.PropDeviceDescription:     prodInfo.ProductName,
	// 	td.WoTTitle:                     node.Name,
	// 	vocab.PropDeviceModel:           prodInfo.Model,
	// 	vocab.PropDeviceHardwareVersion: hwVersion,
	// 	"nodeType":                      node.Type,
	// }
	// it.PubProperties("", props, false)

	return it
}
