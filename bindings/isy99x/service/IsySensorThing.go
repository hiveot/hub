package service

import (
	"errors"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
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

func (it *IsySensorThing) HandleConfigRequest(action *hubclient.ThingMessage) (err error) {
	// TODO: sensor configuration
	return errors.New("unknown config: " + action.Name)
}

// MakeTD returns the TD document representing the node
func (it *IsySensorThing) MakeTD() *tdd.TD {
	td := it.IsyThing.MakeTD()
	// TODO: add sensor properties and events
	return td
}

// NewIsySensorThing creates a ISY sensor device instance.
// Call Init() before use.
func NewIsySensorThing() *IsySensorThing {
	thing := &IsySensorThing{}
	return thing
}
