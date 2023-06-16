package internal

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"time"

	"github.com/hiveot/hub/plugins/owserver/internal/eds"
)

type NodeValueStamp struct {
	timestamp time.Time
	value     string
}

func (binding *OWServerBinding) getPrevValue(nodeID, attrName string) (value NodeValueStamp, found bool) {
	nodeValues, found := binding.values[nodeID]
	if found {
		value, found = nodeValues[attrName]
	}
	return value, found
}

func (binding *OWServerBinding) setPrevValue(nodeID, attrName string, value string) {
	nodeValues, found := binding.values[nodeID]
	if !found {
		nodeValues = make(map[string]NodeValueStamp)
		binding.values[nodeID] = nodeValues
	}
	nodeValues[attrName] = NodeValueStamp{
		timestamp: time.Now(),
		value:     value,
	}
}

// PublishNodeValues publishes node property values of each node
// Properties are combined as submitted as a single 'properties' event.
// Sensor values are send as individual events
func (binding *OWServerBinding) PublishNodeValues(nodes []*eds.OneWireNode) (err error) {

	// Iterate the devices and their properties
	for _, node := range nodes {
		// send all changed property attributes in a single properties event
		attrMap := make(map[string][]byte)
		//thingID := thing.CreateThingID(binding.Config.ID, node.NodeID, node.DeviceType)
		thingID := node.NodeID

		for attrName, attr := range node.Attr {
			// only send the changed values
			prevValue, found := binding.getPrevValue(node.NodeID, attrName)
			age := time.Now().Sub(prevValue.timestamp)
			maxAge := time.Second * time.Duration(binding.Config.RepublishInterval)
			// skip update if the value hasn't changed for less than the republish interval
			skip := found &&
				prevValue.value == attr.Value &&
				age < maxAge

			if !skip {
				binding.setPrevValue(node.NodeID, attrName, attr.Value)
				if attr.IsSensor {
					err = binding.hubClient.PubEvent(thingID, attrName, []byte(attr.Value))
				} else {
					// attribute to be included in the properties event
					attrMap[attrName] = []byte(attr.Value)
				}
			}
		}
		if len(attrMap) > 0 {
			attrMapJSON, _ := json.Marshal(attrMap)
			err = binding.hubClient.PubEvent(thingID, vocab.EventNameProps, attrMapJSON)
		}
	}
	return err
}

// RefreshPropertyValues polls the OWServer hub for changed Thing values
func (binding *OWServerBinding) RefreshPropertyValues() error {
	nodes, err := binding.edsAPI.PollNodes()
	//nodeValueMap, err := binding.PollNodeValues()
	if err == nil {
		err = binding.PublishNodeValues(nodes)
	}
	return err
}
