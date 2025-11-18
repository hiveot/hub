package service

import (
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"

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
		timestamp: time.Now().UTC(),
		value:     value,
	}
}

// PublishNodeValues publishes node property values of each node
// Properties are combined as submitted as a single 'properties' event.
// Only changed properties are included.
// Sensor values are send as individual events.
// All values are sent as text.
func (svc *OWServerBinding) PublishNodeValues(nodes []*eds.OneWireNode, force bool) (err error) {

	evCount := 0
	propCount := 0
	// Iterate the devices and their properties
	for _, node := range nodes {

		// send all changed property attributes in a single properties event
		propMap := make(map[string]any)
		thingID := node.ROMId
		svc.mux.RLock()
		nodeTD, found := svc.things[thingID]
		svc.mux.RUnlock()
		if !found {
			continue
		}

		// inject the writable device title, if set. Default is model name.
		deviceTitleByte, _ := svc.customTitles.Get(thingID)
		if deviceTitleByte != nil && force {
			propMap[wot.WoTTitle] = string(deviceTitleByte)
		}
		// first and unknown values are always changed
		for attrID, attr := range node.Attr {

			// collect changes to property values
			info, found := AttrConfig[attrID]
			if !found {
				// these values are unlisted and ignored
				//slog.Info("Ignored value", "name", attrID, "value", attr.Value)
			} else if info.Ignore {
				// do nothing when ignore is set
			} else {
				if info.IsEvent {
					value, changed := svc.GetValueChange(attrID, attr.Value, info, nodeTD)
					if changed || force {
						slog.Info("PublishThingValues: PubEvent",
							slog.String("thingID", thingID),
							slog.String("attrID", attrID),
							slog.String("value", attr.Value))
						err = svc.ag.PubEvent(nodeTD.ID, attrID, value)
						evCount++
					}
				}
				if info.IsProp {
					value, changed := svc.GetValueChange(attrID, attr.Value, info, nodeTD)
					if changed || force {
						propMap[attrID] = value
					}
				}
			}
		}
		if len(propMap) > 0 {
			slog.Info("PublishThingValues: PubMultipleProperties",
				slog.String("thingID", thingID),
				slog.Int("count", len(propMap)),
			)
			err = svc.ag.PubProperties(thingID, propMap)
			propCount += len(propMap)
		}
	}
	return err
}

// GetValueChange parses the attribute value and track changes to the
// value.
func (svc *OWServerBinding) GetValueChange(
	attrName string, attrValue string, info AttrConversion, td *td.TD) (
	value any, changed bool) {

	var err error

	// parse all data values to their native types and compare if they changed
	// since the previous stored value.
	prevValue, prevFound := svc.getPrevValue(td.ID, attrName)

	switch info.DataType {
	case vocab.WoTDataTypeNumber:
		valueFloat, err2 := strconv.ParseFloat(attrValue, 32)
		prec := math.Pow10(info.Precision)
		roundedInt := int(valueFloat * prec)
		roundedFloat := float64(roundedInt) / prec
		err = err2
		valueDiff := 0.0
		if prevFound {
			valueDiff = math.Abs(roundedFloat - prevValue.value.(float64))
			changed = valueDiff >= info.ChangeNotify
			slog.Debug("GetValueChange", "name", attrName,
				"oldValue", prevValue.value, "newValue", attrValue, "diff", valueDiff)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrName, roundedFloat)
		}
		// return the rounded /truncated result
		value = roundedFloat

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
			svc.setPrevValue(td.ID, attrName, valueInt)
		}
		value = valueInt

	case vocab.WoTDataTypeBool:
		valueBool, err2 := strconv.ParseBool(attrValue)
		err = err2
		if prevFound {
			changed = valueBool != prevValue.value.(bool)
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrName, valueBool)
		}
		value = valueBool
	default: // strings and other values
		value = attrValue
		if prevFound {
			// don't report changes if disabled
			if int(info.ChangeNotify) >= 0 {
				changed = value != prevValue.value.(string)
			}
		} else {
			changed = true
		}
		if changed {
			svc.setPrevValue(td.ID, attrName, value)
		}
	}
	if err != nil {
		slog.Error("value conversion error", "err", err.Error)
	}
	return value, changed
}

// RefreshPropertyValues polls the OWServer hub for changed Thing values
// Set 'force' to force publishing values event when not changed
func (svc *OWServerBinding) RefreshPropertyValues(force bool) error {
	nodes, err := svc.edsAPI.PollNodes()

	if err == nil {

		err = svc.PublishNodeValues(nodes, force)
	}
	return err
}
