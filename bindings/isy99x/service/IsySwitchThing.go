package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/lib/agent"
	"github.com/hiveot/hub/lib/messaging"
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
func (it *IsySwitchThing) HandleActionRequest(
	ag *agent.Agent, req *messaging.RequestMessage) *messaging.ResponseMessage {
	var restPath = ""
	var newValue = ""
	var input bool
	var output bool

	// FIXME: req keys are the raw keys, not @type
	// supported actions: on, off
	if req.Name == "ST" {
		input = utils.DecodeAsBool(req.Input)
		newValue = "DOF"
		if input {
			newValue = "DON"
		}
	} else {
		// unknown req
		newValue = ""
		err := fmt.Errorf("HandleRequest. Unknown req: %s", req.Name)
		return req.CreateResponse(nil, err)
	}

	// Post a new value
	restPath = fmt.Sprintf("/rest/nodes/%s/cmd/%s", it.nodeID, newValue)
	err := it.isyAPI.SendRequest("GET", restPath, "", nil)

	// read the result. As this takes a while, retry every second for 5 seconds
	if err != nil {
		return req.CreateResponse(nil, err)
	}
	// return a 'running' status while reading back the result
	//send a 'running' ActionStatus message
	//ag.PubActionStatus(req)

	// in the background poll for status update until completed
	go func() {
		var resp *messaging.ResponseMessage
		hasUpdated := false
		// TODO: clean this up. Use websocket instead of repeated polling.
		for i := 0; i < 5; i++ {
			nodeInfo, err := it.isyAPI.ReadNodeInfo(it.nodeID)
			if err == nil {
				// TODO: repeat a few times in the background and send a response
				// last response is completed
				time.Sleep(time.Millisecond) // wait for processing request
				slog.Info("Switch action (waiting for change)",
					slog.Bool("input", input),
					slog.String("current output", nodeInfo.Properties.Property.Value),
				)
				// on/off returns 0 when off
				output = (nodeInfo.Properties.Property.Value != "0")
				if output == input {
					// confirmed property change
					_ = it.HandleValueUpdate(req.Name,
						nodeInfo.Properties.Property.UOM,
						nodeInfo.Properties.Property.Value)
					hasUpdated = true
					break
				}
			}
			time.Sleep(time.Second)
		}
		if !hasUpdated {
			// no update, consider this failed
			err = fmt.Errorf("No response from device")
			resp = req.CreateResponse(output, err)
		} else {
			// completed
			resp = req.CreateResponse(output, nil)
			// FIXME: send notification in the background for all subscribers.
			//go it.PublishAllThingValues()
		}
		_ = ag.GetConnection().SendResponse(resp)
	}()
	// send the response async when done
	return nil
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
	tdi := it.IsyThing.MakeTD()
	// value of switch property ID "ST" is "0" or "255"
	// TODO: support for switch events
	//td.AddEvent("ST", "On/Off", "",
	//	&tdd.DataSchema{Type: vocab.DataTypeBool}).
	//	SetAtType(vocab.ActionSwitchOnOff)

	action := tdi.AddAction("ST", "Switch on/off", "",
		&td.DataSchema{
			AtType: vocab.ActionSwitchOnOff,
			Type:   wot.DataTypeBool,
			Enum:   []interface{}{"on", "off"},
		})
	// output data same as input
	action.Output = action.Input

	// add a corresponding property for the switch state
	tdi.AddProperty("ST", "Switch on/off", "On/Off switch", wot.DataTypeBool)

	//td.AddSwitchAction(vocab.ActionSwitchOff, "Switch off")
	//td.AddSwitchAction(vocab.ActionSwitchToggle, "Toggle switch")

	return tdi
}

// NewIsySwitchThing creates a new instance of an ISY switch.
// Call Init() before use
func NewIsySwitchThing(evHandler IsyEventHandler) *IsySwitchThing {
	thing := &IsySwitchThing{IsyThing{evHandler: evHandler}}
	return thing
}
