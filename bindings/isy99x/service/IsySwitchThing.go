package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/lib/things"
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

func (it *IsySwitchThing) GetTD() *things.TD {
	td := it.IsyThing.GetTD()
	// value of switch property ID "ST" is "0" or "255"
	td.AddEvent("ST", vocab.PropSwitchOnOff, "On/Off", "",
		&things.DataSchema{Type: vocab.WoTDataTypeBool})

	// AddSwitchEvent is short for adding an event for a switch
	//td.AddSwitchEvent(vocab.PropSwitchOnOff, "On/Off changed")
	td.AddSwitchAction(vocab.ActionSwitchOn, "Switch on")
	td.AddSwitchAction(vocab.ActionSwitchOff, "Switch off")
	td.AddSwitchAction(vocab.ActionSwitchToggle, "Toggle switch")

	return td
}

//func (it *IsySwitchThing) HandleConfigRequest(tv *things.ThingMessage) (err error) {
//	err = it.IsyThing.HandleConfigRequest(tv)
//	return err
//}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsySwitchThing) HandleActionRequest(action *things.ThingMessage) (err error) {
	var restPath = ""
	var newValue = ""
	// FIXME: action keys are the raw keys, not @type
	// supported actions: on, off
	if action.Key == vocab.ActionSwitchOn {
		newValue = "DON"
	} else if action.Key == vocab.ActionSwitchOff {
		newValue = "DOF"
	} else if action.Key == vocab.ActionSwitchToggle {
		newValue = "DOF"
		oldValue, found := it.propValues.GetValue(action.Key)
		if !found || oldValue == "DOF" {
			newValue = "DON"
		}
	} else {
		// unknown action
		newValue = ""
		err = fmt.Errorf("HandleActionRequest. Unknown action: %s", action.Key)
		return err
	}

	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
	err = it.isyAPI.SendRequest("GET", restPath, "", nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		err = it.HandleValueUpdate(action.Key, "", newValue)
		//it.currentProps[actionID] = newValue
	}
	return err
}

func (it *IsySwitchThing) HandleValueUpdate(propID string, uom string, newValue string) error {
	it.mux.Lock()
	defer it.mux.Unlock()
	// convert the switch value to a boolean
	if propID == "ST" {
		boolValue := true
		if newValue == "" || newValue == "0" {
			boolValue = false
		}
		it.propValues.SetValueBool(propID, boolValue)
	} else {
		it.propValues.SetValue(propID, newValue)
	}
	return nil
}

// Init initializes the IsyThing base class
// This determines the device type from prodInfo and sets property values for
// product and model.
func (it *IsySwitchThing) Init(ic *isy.IsyAPI, thingID string, node *isy.IsyNode, prodInfo InsteonProduct, hwVersion string) {
	it.IsyThing.Init(ic, thingID, node, prodInfo, hwVersion)
}

// NewIsySwitchThing creates a new instance of an ISY switch.
// Call Init() before use
func NewIsySwitchThing() *IsySwitchThing {
	thing := &IsySwitchThing{}
	return thing
}
