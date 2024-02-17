package service

import (
	"errors"
	"github.com/hiveot/hub/lib/things"
)

// IsySensorThing is a general-purpose sensor
type IsySensorThing struct {
	NodeThing
}

func (it *IsySensorThing) HandleConfigRequest(tv *things.ThingValue) (err error) {
	// TODO: sensor configuration
	return errors.New("unknown config: " + tv.Name)
}

// GetTD returns the TD document representing the node
//func (t *IsySensorThing) GetTD() *things.TD {
//
//}

// NewIsySensorThing creates a ISY sensor device instance.
// Call Init() before use.
func NewIsySensorThing() *IsySensorThing {
	thing := &IsySensorThing{}
	thing.configHandler = thing.HandleConfigRequest
	return thing
}
