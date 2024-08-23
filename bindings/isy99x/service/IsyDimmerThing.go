package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
)

// IsyDimmerThing is a general-purpose dimmer switch
type IsyDimmerThing struct {
	IsyThing
}

// GetValues returns the property and event values for publication
func (it *IsyDimmerThing) GetValues(onlyChanges bool) map[string]any {
	propValues := it.IsyThing.GetPropValues(onlyChanges)
	propValues[vocab.PropSwitchDimmer] = propValues[vocab.PropSwitchDimmer]
	return propValues
}

func (it *IsyDimmerThing) GetTD() *tdd.TD {
	td := it.IsyThing.GetTD()
	// AddSwitchEvent is short for adding an event for a switch
	td.AddDimmerEvent(vocab.PropSwitchDimmer)

	a := td.AddDimmerAction(vocab.ActionDimmerSet)
	a.Input.Unit = vocab.UnitPercent
	a.Input.Minimum = 0
	a.Input.Maximum = 100
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

func (it *IsyDimmerThing) HandleConfigRequest(action *hubclient.ThingMessage) (err error) {
	return errors.New("unknown config: " + action.Key)
}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsyDimmerThing) HandleActionRequest(action *hubclient.ThingMessage) (err error) {
	var restPath = ""
	var newValue = ""
	// FIXME: action keys are node attributes keys, not vocab @types (or are they?)
	// supported actions: on, off
	if action.Key == vocab.ActionDimmerSet {
		newValue = action.DataAsText()
		restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)

		//} else if action.Name == vocab.VocabActionDecrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
		//} else if action.Name == vocab.VocabActionIncrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
	} else {
		// unknown action
		err = fmt.Errorf("HandleActionRequest. Unknown action: '%s'", action.Key)
		return err
	}

	err = it.isyAPI.SendRequest("GET", restPath, "", nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		err = it.HandleValueUpdate(action.Key, "", newValue)
	}

	return err
}

// NewIsyDimmerThing creates a new instance of an ISY dimmer.
// Call Init() before use
func NewIsyDimmerThing() *IsyDimmerThing {
	thing := &IsyDimmerThing{}
	return thing
}
