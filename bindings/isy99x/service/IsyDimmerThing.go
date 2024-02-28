package service

import (
	"errors"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
)

// IsyDimmerThing is a general-purpose dimmer switch
type IsyDimmerThing struct {
	NodeThing
}

func (it *IsyDimmerThing) GetTD() *things.TD {
	td := it.NodeThing.GetTD()
	// AddSwitchEvent is short for adding an event for a switch
	td.AddDimmerEvent(vocab.PropSwitchDimmer)

	a := td.AddDimmerAction(vocab.ActionDimmerSet)
	a.Input.Unit = vocab.UnitPercent
	a.Input.NumberMinimum = 0
	a.Input.NumberMaximum = 100
	// TODO: increment and decrement
	//a = td.AddDimmerAction(vocab.VocabActionDecrement)
	//a.Input.Unit = vocab.UnitNamePercent
	//a.Input.NumberMinimum = 0
	//a.Input.NumberMaximum = 100
	//a = td.AddDimmerAction(vocab.VocabActionIncrement)
	//a.Input.Unit = vocab.UnitNamePercent
	//a.Input.NumberMinimum = 0
	//a.Input.NumberMaximum = 100

	return td
}

func (it *IsyDimmerThing) HandleConfigRequest(tv *things.ThingValue) (err error) {
	return errors.New("unknown config: " + tv.Name)
}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsyDimmerThing) HandleActionRequest(tv *things.ThingValue) (err error) {
	var restPath = ""
	var newValue = ""
	// supported actions: on, off
	if tv.Name == vocab.ActionDimmerSet {
		newValue = string(tv.Data)
		restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.id, newValue)

		//} else if tv.Name == vocab.VocabActionDecrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.id, newValue)
		//} else if tv.Name == vocab.VocabActionIncrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.id, newValue)
	} else {
		// unknown action
		err = fmt.Errorf("HandleActionRequest. Unknown action: '%s'", tv.Name)
		return err
	}

	err = it.ic.SendRequest("GET", restPath, nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		err = it.HandleValueUpdate(tv.Name, "", newValue)
	}

	return err
}

// NewIsyDimmerThing creates a new instance of an ISY dimmer.
// Call Init() before use
func NewIsyDimmerThing() *IsyDimmerThing {
	thing := &IsyDimmerThing{}
	thing.actionHandler = thing.HandleActionRequest
	thing.configHandler = thing.HandleConfigRequest
	return thing
}
