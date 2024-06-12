package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/hiveot/hub/bindings/owserver/service/eds"
)

type NodeValueStamp struct {
	timestamp time.Time
	value     any
}

func (svc *OWServerBinding) getPrevValue(nodeID, attrName string) (value NodeValueStamp, found bool) {
	nodeValues, found := svc.values[nodeID]
	if found {
		value, found = nodeValues[attrName]
	}
	return value, found
}

func (svc *OWServerBinding) setPrevValue(nodeID, attrName string, value any) {
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
// Only changed properties are included.
// Sensor values are send as individual events.
// All values are sent as text.
func (svc *OWServerBinding) PublishNodeValues(nodes []*eds.OneWireNode, force bool) (err error) {

	// Iterate the devices and their properties
	for _, node := range nodes {
		// send all changed property attributes in a single properties event
		propMap := make(map[string]string)
		//thingID := things.CreateThingID(svc.config.ID, node.NodeID, node.DeviceType)
		thingID := node.ROMId
		svc.mux.RLock()
		nodeTD, found := svc.things[thingID]
		svc.mux.RUnlock()
		if !found {
			continue
		}

		for attrID, attr := range node.Attr {
			// collect changes to property values
			info, found := AttrConfig[attrID]
			if found && info.Ignore {
				// do nothing when ignore is set
			} else if found && info.IsEvent {
				valueStr, changed := svc.GetValueChange(attrID, attr.Value, info, nodeTD)
				if changed || force {
					err = svc.hc.PubEvent(nodeTD.ID, attrID, []byte(valueStr))
				}
			} else if !found || info.IsProp {
				// first and unknown values are always changed
				valueStr, changed := svc.GetValueChange(attrID, attr.Value, info, nodeTD)
				if changed || force {
					propMap[attrID] = valueStr
				}
			}
		}
		if len(propMap) > 0 {
			err = svc.hc.PubProps(thingID, propMap)
		}
	}
	return err
}

// GetValueChange formats the attribute value and track changes to the
// value using the attribute conversion settings.
func (svc *OWServerBinding) GetValueChange(
	attrKey string, attrValue string, info AttrConversion, td *things.TD) (
	strValue string, changed bool) {

	var err error

	// parse all data values to their native types and compare if they changed
	// since the previous stored value.
	prevValue, prevFound := svc.getPrevValue(td.ID, attrKey)

	switch info.DataType {
	case vocab.WoTDataTypeNumber:
		valueFloat, err2 := strconv.ParseFloat(attrValue, 32)
		err = err2
		valueDiff := 0.0
		if prevFound {
			valueDiff = math.Abs(valueFloat - prevValue.value.(float64))
			changed = valueDiff >= info.ChangeNotify
			slog.Debug("GetValueChange", "key", attrKey,
				"oldValue", prevValue.value, "newValue", attrValue, "diff", valueDiff)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrKey, valueFloat)
		}
		// return the formatted result
		strValue = strconv.FormatFloat(valueFloat, 'f', info.Precision, 32)

	case vocab.WoTDataTypeInteger, vocab.WoTDataTypeUnsignedInt:
		valueInt64, err2 := strconv.ParseInt(attrValue, 10, 32)
		valueInt := int(valueInt64)
		err = err2
		valueDiff := 0
		if prevFound {
			valueDiff = valueInt - prevValue.value.(int)
			if valueDiff < 0 {
				valueDiff = -valueDiff
			}
			changed = valueDiff >= int(info.ChangeNotify)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrKey, valueInt)
		}
		strValue = strconv.FormatInt(valueInt64, 10)

	case vocab.WoTDataTypeBool:
		valueBool, err2 := strconv.ParseBool(attrValue)
		err = err2
		if prevFound {
			changed = valueBool != prevValue.value.(bool)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrKey, valueBool)
		}
		strValue = strconv.FormatBool(valueBool)
	default: // strings and other values
		strValue = attrValue
		if prevFound {
			changed = strValue != prevValue.value.(string)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrKey, strValue)
		}
	}
	if err != nil {
		slog.Error("value conversion error", "err", err.Error)
	}
	return strValue, changed
}

// RefreshPropertyValues polls the OWServer hub for changed Thing values
func (svc *OWServerBinding) RefreshPropertyValues() error {
	nodes, err := svc.edsAPI.PollNodes()
	//nodeValueMap, err := svc.PollNodeValues()
	if err == nil {
		err = svc.PublishNodeValues(nodes, false)
	}
	return err
}
