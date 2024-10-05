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

// PropertyListToMap creates a map from a digitwin property list
func PropertyListToMap(valueList []digitwin.PropertyValue) map[string]digitwin.PropertyValue {
	valMap := make(map[string]digitwin.PropertyValue)
	for _, value := range valueList {
		valMap[value.Name] = value
	}
	return valMap
}

// EventListToMap creates a map from a digitwin event list
func EventListToMap(valueList []digitwin.EventValue) map[string]digitwin.EventValue {
	valMap := make(map[string]digitwin.EventValue)
	for _, value := range valueList {
		valMap[value.Name] = value
	}
	return valMap
}
