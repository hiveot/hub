package service

import (
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/ser"
	"time"

	"github.com/hiveot/hub/bindings/owserver/service/eds"
)

type NodeValueStamp struct {
	timestamp time.Time
	value     string
}

func (svc *OWServerBinding) getPrevValue(nodeID, attrName string) (value NodeValueStamp, found bool) {
	nodeValues, found := svc.values[nodeID]
	if found {
		value, found = nodeValues[attrName]
	}
	return value, found
}

func (svc *OWServerBinding) setPrevValue(nodeID, attrName string, value string) {
	nodeValues, found := svc.values[nodeID]
	if !found {
		nodeValues = make(map[string]NodeValueStamp)
		svc.values[nodeID] = nodeValues
	}
	nodeValues[attrName] = NodeValueStamp{
		timestamp: time.Now(),
		value:     value,
	}
}

// PublishNodeValues publishes node property values of each node
// Properties are combined as submitted as a single 'properties' event.
// Sensor values are send as individual events.
// All values are sent as text.
func (svc *OWServerBinding) PublishNodeValues(nodes []*eds.OneWireNode) (err error) {

	// Iterate the devices and their properties
	for _, node := range nodes {
		// send all changed property attributes in a single properties event
		attrMap := make(map[string]string)
		//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
		thingID := node.ROMId

		for attrID, attr := range node.Attr {
			// only send the changed values
			prevValue, found := svc.getPrevValue(node.ROMId, attrID)
			age := time.Now().Sub(prevValue.timestamp)
			maxAge := time.Second * time.Duration(svc.config.RepublishInterval)
			// skip update if the value hasn't changed for less than the republish interval
			skip := found && prevValue.value == attr.Value && age < maxAge

			if !skip {
				svc.setPrevValue(node.ROMId, attrID, attr.Value)
				sensorInfo, isSensor := SensorAttrVocab[attrID]
				_ = sensorInfo
				if isSensor {
					err = svc.hc.PubEvent(thingID, attrID, []byte(attr.Value))
				} else {
					// attribute to be included in the properties event
					attrMap[attrID] = attr.Value
				}
			}
		}
		if len(attrMap) > 0 {
			attrMapJSON, _ := ser.Marshal(attrMap)
			err = svc.hc.PubEvent(thingID, transports.EventNameProps, attrMapJSON)
		}
	}
	return err
}

// RefreshPropertyValues polls the OWServer hub for changed Thing values
func (svc *OWServerBinding) RefreshPropertyValues() error {
	nodes, err := svc.edsAPI.PollNodes()
	//nodeValueMap, err := svc.PollNodeValues()
	if err == nil {
		err = svc.PublishNodeValues(nodes)
	}
	return err
}
