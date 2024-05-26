package service

import (
	"errors"
	"github.com/hiveot/hub/lib/things"
)

// IsySensorThing is a general-purpose sensor
type IsySensorThing struct {
	IsyThing
}

// GetPropValues returns the property and event values for publication
func (it *IsySensorThing) GetPropValues(onlyChanges bool) map[string]string {
	propValues := it.IsyThing.GetPropValues(onlyChanges)
	return propValues
}

// GetTD returns the TD document representing the node
func (it *IsySensorThing) GetTD() *things.TD {
	td := it.IsyThing.GetTD()
	// TODO: add sensor properties and events
	return td
}

func (it *IsySensorThing) HandleConfigRequest(action *things.ThingMessage) (err error) {
	// TODO: sensor configuration
	return errors.New("unknown config: " + action.Key)
}

// NewIsySensorThing creates a ISY sensor device instance.
// Call Init() before use.
func NewIsySensorThing() *IsySensorThing {
	thing := &IsySensorThing{}
	return thing
}
