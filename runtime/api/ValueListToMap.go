package api

import "github.com/hiveot/hub/api/go/digitwin"

// ActionListToMap creates a map from a digitwin action value list
func ActionListToMap(valueList []digitwin.ActionValue) map[string]digitwin.ActionValue {
	valMap := make(map[string]digitwin.ActionValue)
	for _, value := range valueList {
		valMap[value.Name] = value
	}
	return valMap
}

// ValueListToMap creates a map from a digitwin value list
func ValueListToMap(valueList []digitwin.ThingValue) map[string]digitwin.ThingValue {
	valMap := make(map[string]digitwin.ThingValue)
	for _, value := range valueList {
		valMap[value.Name] = value
	}
	return valMap
}
