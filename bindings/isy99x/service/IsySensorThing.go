package service

import (
	"errors"

	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/lib/messaging"
)

// IsySensorThing is a general-purpose sensor
type IsySensorThing struct {
	IsyThing
}

// GetPropValues returns the property and event values for publication
func (it *IsySensorThing) GetPropValues(onlyChanges bool) map[string]any {
	propValues := it.IsyThing.GetPropValues(onlyChanges)
	return propValues
}

func (it *IsySensorThing) HandleConfigRequest(req *messaging.RequestMessage) *messaging.ResponseMessage {
	// TODO: sensor configuration
	return req.CreateResponse(nil, errors.New("unknown config: "+req.Name))
}

// MakeTD returns the TD document representing the node
func (it *IsySensorThing) MakeTD() *td.TD {
	td := it.IsyThing.MakeTD()
	// TODO: add sensor properties and events
	return td
}

// NewIsySensorThing creates a ISY sensor device instance.
// Call Init() before use.
func NewIsySensorThing(evHandler IsyEventHandler) *IsySensorThing {
	thing := &IsySensorThing{IsyThing{evHandler: evHandler}}
	return thing
}
