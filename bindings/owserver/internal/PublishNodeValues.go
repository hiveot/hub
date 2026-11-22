package internal

// type NodeValueStamp struct {
// 	timestamp time.Time
// 	value     any
// }

// func (svc *OWServerGateway) getPrevValue(nodeID, attrName string) (value NodeValueStamp, found bool) {
// 	svc.mux.RLock()
// 	defer svc.mux.RUnlock()
// 	nodeValues, found := svc.values[nodeID]
// 	if found {
// 		value, found = nodeValues[attrName]
// 	}
// 	return value, found
// }

// func (svc *OWServerGateway) setPrevValue(nodeID, attrName string, value any) {
// 	svc.mux.Lock()
// 	defer svc.mux.Unlock()
// 	nodeValues, found := svc.values[nodeID]
// 	if !found {
// 		nodeValues = make(map[string]NodeValueStamp)
// 		svc.values[nodeID] = nodeValues
// 	}
// 	nodeValues[attrName] = NodeValueStamp{
// 		timestamp: time.Now().UTC(),
// 		value:     value,
// 	}
// }

// GetValueChange parses the attribute value and track changes to the
// value.
// func (svc *OWServerGateway) GetValueChange(
// 	attrName string, attrValue string, info AttrConversion, tdDoc *wot.TD) (
// 	value any, changed bool) {

// 	var err error

// 	// parse all data values to their native types and compare if they changed
// 	// since the previous stored value.
// 	prevValue, prevFound := svc.getPrevValue(tdDoc.ID, attrName)

// 	switch info.DataType {
// 	case wot.DataTypeNumber:
// 		valueFloat, err2 := strconv.ParseFloat(attrValue, 32)
// 		prec := math.Pow10(info.Precision)
// 		roundedInt := int(valueFloat * prec)
// 		roundedFloat := float64(roundedInt) / prec
// 		err = err2
// 		valueDiff := 0.0
// 		if prevFound {
// 			valueDiff = math.Abs(roundedFloat - prevValue.value.(float64))
// 			changed = valueDiff >= info.ChangeNotify
// 			slog.Debug("GetValueChange", "name", attrName,
// 				"oldValue", prevValue.value, "newValue", attrValue, "diff", valueDiff)
// 		} else {
// 			changed = true
// 		}
// 		if changed {
// 			svc.setPrevValue(tdDoc.ID, attrName, roundedFloat)
// 		}
// 		// return the rounded /truncated result
// 		value = roundedFloat

// 	case wot.DataTypeInteger, wot.DataTypeUnsignedInt:
// 		valueInt64, err2 := strconv.ParseInt(attrValue, 10, 32)
// 		valueInt := int(valueInt64)
// 		err = err2
// 		valueDiff := 0
// 		if prevFound {
// 			valueDiff = valueInt - prevValue.value.(int)
// 			if valueDiff < 0 {
// 				valueDiff = -valueDiff
// 			}
// 			changed = valueDiff >= int(info.ChangeNotify)
// 		} else {
// 			changed = true
// 		}
// 		if changed {
// 			svc.setPrevValue(tdDoc.ID, attrName, valueInt)
// 		}
// 		value = valueInt

// 	case wot.DataTypeBool:
// 		valueBool, err2 := strconv.ParseBool(attrValue)
// 		err = err2
// 		if prevFound {
// 			changed = valueBool != prevValue.value.(bool)
// 		} else {
// 			changed = true
// 		}
// 		if changed {
// 			svc.setPrevValue(tdDoc.ID, attrName, valueBool)
// 		}
// 		value = valueBool
// 	default: // strings and other values
// 		value = attrValue
// 		if prevFound {
// 			// don't report changes if disabled
// 			if int(info.ChangeNotify) >= 0 {
// 				changed = value != prevValue.value.(string)
// 			}
// 		} else {
// 			changed = true
// 		}
// 		if changed {
// 			svc.setPrevValue(tdDoc.ID, attrName, value)
// 		}
// 	}
// 	if err != nil {
// 		slog.Error("value conversion error", "err", err.Error)
// 	}
// 	return value, changed
// }

// RefreshPropertyValues polls the OWServer hub for changed Thing values
// Set 'onlyChanges' to only publish changed values.
// func (svc *OWServerGateway) UpdateThingValues(onlyChanges bool) error {
// 	nodes, err := svc.edsAPI.PollNodes()

// 	if err == nil {
// 		for _, node := range nodes {
// 			thingID := node.ROMId
// 			owt, found := svc.owThings[thingID]
// 			if found {
// 				title, err := svc.customTitles.Get(thingID)
// 				titleStr := ""
// 				if err == nil && title != nil {
// 					titleStr = string(title)
// 				}
// 				owt.UpdateProperties(node, titleStr, onlyChanges)
// 			}
// 		}
// 	}
// 	return err
// }
