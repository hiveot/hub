package api

import (
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
)

// ActionListToMap creates a map from a digitwin action value list
func ActionListToMap(valueList []digitwin.ActionStatus) map[string]digitwin.ActionStatus {
	valMap := make(map[string]digitwin.ActionStatus)
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
