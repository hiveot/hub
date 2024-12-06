package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils"
)

// IsySwitchThing is a general-purpose on/off switch
type IsySwitchThing struct {
	IsyThing
}

// GetPropValues returns the property and event values for publication
func (it *IsySwitchThing) GetPropValues(onlyChanges bool) map[string]any {
	propValues := it.IsyThing.GetPropValues(onlyChanges)
	return propValues
}

//func (it *IsySwitchThing) HandleConfigRequest(tv *things.ThingMessage) (err error) {
//	err = it.IsyThing.HandleConfigRequest(tv)
//	return err
//}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsySwitchThing) HandleActionRequest(action *transports.ThingMessage) (err error) {
	var restPath = ""
	var newValue = ""
	// FIXME: action keys are the raw keys, not @type
	// supported actions: on, off
	if action.Name == "ST" {
		newValueBool := utils.DecodeAsBool(action.Data)
		newValue = "DOF"
		if newValueBool {
			newValue = "DON"
		}
		//} else if action.Name == vocab.ActionSwitchToggle {
		//	newValue = "DOF"
		//	oldValue, found := it.propValues.GetValue(action.Name)
		//	if !found || oldValue == "DOF" {
		//		newValue = "DON"
		//	}
	} else {
		// unknown action
		newValue = ""
		err = fmt.Errorf("HandleActionRequest. Unknown action: %s", action.Name)
		return err
	}

	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
	err = it.isyAPI.SendRequest("GET", restPath, "", nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		//err = it.HandleValueUpdate(action.Name, "", newValue)
	}
	return err
}

// HandleValueUpdate receives a new value for the given property
func (it *IsySwitchThing) HandleValueUpdate(propName string, uom string, newValue string) error {
	it.mux.Lock()
	defer it.mux.Unlock()
	// convert the switch value to a boolean
	if propName == "ST" {
		boolValue := false
		if newValue == "1" || newValue == "255" {
			boolValue = true
		}
		it.propValues.SetValueBool(propName, boolValue)
	} else {
		it.propValues.SetValue(propName, newValue)
	}
	return nil
}

// Init initializes the IsyThing base class
// This determines the device type from prodInfo and sets property values for
// product and model.
func (it *IsySwitchThing) Init(ic *isy.IsyAPI, thingID string, node *isy.IsyNode, prodInfo InsteonProduct, hwVersion string) {
	it.IsyThing.Init(ic, thingID, node, prodInfo, hwVersion)
}

func (it *IsySwitchThing) MakeTD() *td.TD {
	td := it.IsyThing.MakeTD()
	// value of switch property ID "ST" is "0" or "255"
	// TODO: support for switch events
	//td.AddEvent("ST", "On/Off", "",
	//	&tdd.DataSchema{Type: vocab.WoTDataTypeBool}).
	//	SetAtType(vocab.ActionSwitchOnOff)

	td.AddAction("ST", "Switch on/off", "",
		&td.DataSchema{
			AtType: vocab.ActionSwitchOnOff,
			Type:   wot.WoTDataTypeBool,
			Enum:   []interface{}{"on", "off"},
		}).SetAtType(vocab.ActionSwitchOnOff)

	//td.AddSwitchAction(vocab.ActionSwitchOff, "Switch off")
	//td.AddSwitchAction(vocab.ActionSwitchToggle, "Toggle switch")

	return td
}

// NewIsySwitchThing creates a new instance of an ISY switch.
// Call Init() before use
func NewIsySwitchThing(evHandler IsyEventHandler) *IsySwitchThing {
	thing := &IsySwitchThing{IsyThing{evHandler: evHandler}}
	return thing
}
