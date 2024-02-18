package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
)

// IsySwitchThing is a general-purpose on/off switch
type IsySwitchThing struct {
	NodeThing
}

func (it *IsySwitchThing) GetTD() *things.TD {
	td := it.NodeThing.GetTD()
	// AddSwitchEvent is short for adding an event for a switch
	td.AddSwitchEvent(vocab.VocabOnOffSwitch)
	td.AddSwitchAction(vocab.VocabActionOn)
	td.AddSwitchAction(vocab.VocabActionOff)

	return td
}

func (it *IsySwitchThing) HandleConfigRequest(tv *things.ThingValue) (err error) {
	return errors.New("unknown config: " + tv.Name)
}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsySwitchThing) HandleActionRequest(tv *things.ThingValue) (err error) {
	var restPath = ""
	var newValue = ""
	// supported actions: on, off
	if tv.Name == vocab.VocabActionOn {
		newValue = "DON"
	} else if tv.Name == vocab.VocabActionOff {
		newValue = "DOF"
	} else if tv.Name == vocab.VocabActionToggle {
		newValue = "DOF"
		oldValue, found := it.propValues.GetValue(tv.Name)
		if !found || oldValue == "DOF" {
			newValue = "DON"
		}
	} else {
		// unknown action
		newValue = ""
		err = fmt.Errorf("HandleActionRequest. Unknown action: %s", tv.Name)
		return err
	}

	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.id, newValue)
	err = it.ic.SendRequest("GET", restPath, nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		err = it.HandleValueUpdate(tv.Name, "", newValue)
		//it.currentProps[actionID] = newValue
	}
	return err
}

// NewIsySwitchThing creates a new instance of an ISY switch.
// Call Init() before use
func NewIsySwitchThing() *IsySwitchThing {
	thing := &IsySwitchThing{}
	thing.actionHandler = thing.HandleActionRequest
	thing.configHandler = thing.HandleConfigRequest
	return thing
}
