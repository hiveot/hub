package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
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

func (it *IsyDimmerThing) MakeTD() *td.TD {
	tdi := it.IsyThing.MakeTD()
	// AddSwitchEvent is short for adding an event for a switch
	// TODO: add dimmer change events
	//td.AddDimmerEvent(vocab.PropSwitchDimmer)

	a := tdi.AddAction(vocab.ActionDimmerSet, "Set Dimmer", "",
		&td.DataSchema{Type: wot.WoTDataTypeInteger},
	)
	a.SetAtType(vocab.ActionDimmer)

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

	return tdi
}

func (it *IsyDimmerThing) HandleConfigRequest(req transports.RequestMessage) transports.ResponseMessage {
	return req.CreateResponse(nil, errors.New("unknown config: "+req.Name))
}

// HandleActionRequest handles request to execute an action on this device
// actionID string as defined in the action affordance
// newValue is not used as these actions do not carry a parameter
func (it *IsyDimmerThing) HandleActionRequest(req transports.RequestMessage) transports.ResponseMessage {
	var restPath = ""
	var newValue = ""
	// FIXME: req keys are node attributes keys, not vocab @types (or are they?)
	// supported actions: on, off
	if req.Name == vocab.ActionDimmerSet {
		newValue = req.ToString()
		restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)

		//} else if req.Name == vocab.VocabActionDecrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
		//} else if req.Name == vocab.VocabActionIncrement {
		//	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
	} else {
		// unknown req
		err := fmt.Errorf("HandleRequest. Unknown req: '%s'", req.Name)
		return req.CreateResponse(nil, err)
	}

	err := it.isyAPI.SendRequest("GET", restPath, "", nil)
	if err == nil {
		// TODO: handle event from gateway using websockets. For now just assume this worked.
		err = it.HandleValueUpdate(req.Name, "", newValue)
	}
	return req.CreateResponse(nil, err)
}

// NewIsyDimmerThing creates a new instance of an ISY dimmer.
// Call Init() before use
func NewIsyDimmerThing(evHandler IsyEventHandler) *IsyDimmerThing {
	thing := &IsyDimmerThing{IsyThing{evHandler: evHandler}}
	return thing
}
