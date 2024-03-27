package service

import "github.com/hiveot/hub/lib/things"

//Digital Twin

// IDigiTwin defines the interface of a digital twin of a Thing
//
// Interface of a Thing digital twin instance
type IDigiTwin interface {
	// GetID returns the ID of the Thing and its agent
	GetID() (agentID string, thingID string)

	// GetProps returns the latest property values
	GetProps() map[string]string

	// GetTD returns the TD of the digital twin
	GetTD() *things.TD

	// LoadEvent loads a new event value of a digital twin as provided by the thing agent
	LoadEvent(eventName string, value string) error

	// LoadProps load new property values of a digital twin as provided by the thing agent
	LoadProps(props map[string]string) error

	// LoadTD updates the TD of the digital twin's as provided by the thing agent
	// This updates the TD stored on the digital twin.
	// The TD is modified with forms for communication via the Hub.
	LoadTD(td *things.TD)

	// RequestAction requests an action on the thing
	RequestAction(name string, payload []byte) ([]byte, error)

	// RequestConfig requests a configuration change of the thing
	RequestConfig(propName string, value string) error
}
