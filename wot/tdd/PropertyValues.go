package tdd

import (
	"strconv"
	"sync"
)

// PropertyValues is a helper type for managing property values and their changes
// This is concurrent safe.
type PropertyValues struct {
	latest  map[string]any
	changed map[string]any
	mux     sync.RWMutex
}

// GetValue returns the string representation of the latest value for the given name
// If the name doesn't exist this returns an empty string
func (pv *PropertyValues) GetValue(name string) (value any, found bool) {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	value, found = pv.latest[name]
	return value, found
}

// GetValues returns the latest or changed values and reset the changed values
// If onlyChanges is set then only return the changes
func (pv *PropertyValues) GetValues(onlyChanges bool) (v map[string]any) {
	if onlyChanges {
		v = pv.GetChanged(true)
	} else {
		v = pv.GetLatest()
	}
	pv.changed = make(map[string]any)
	return v
}

// GetLatest returns the latest map of properties
// This does not reset the 'changed' values so it can be called as often as needed.
func (pv *PropertyValues) GetLatest() map[string]any {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	latestValues := make(map[string]any)
	for k, v := range pv.latest {
		latestValues[k] = v
	}
	return latestValues
}

// HasChanges returns whether there are changed properties
func (pv *PropertyValues) HasChanges() bool {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	return len(pv.changed) > 0
}

// GetChanged returns a copy of the map of changed property values
// clear will clear the changed values
func (pv *PropertyValues) GetChanged(clear bool) map[string]any {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	changedValues := pv.changed
	if clear {
		pv.changed = make(map[string]any)
	} else {
		changedValues = make(map[string]any)
		for k, v := range pv.changed {
			changedValues[k] = v
		}
	}
	return changedValues
}

// SetValue update the property 'name' with a string value
// Returns true if changed.
func (pv *PropertyValues) SetValue(name string, newValue any) {
	pv.mux.Lock()
	defer pv.mux.Unlock()
	oldValue, found := pv.latest[name]
	if !found || oldValue != newValue {
		pv.latest[name] = newValue
		pv.changed[name] = newValue
	}
}

// SetValueBool sets the boolean true/false value in the property map
func (pv *PropertyValues) SetValueBool(name string, newValue bool) {
	valueString := "false"
	if newValue {
		valueString = "true"
	}
	pv.SetValue(name, valueString)
}

// SetValueInt sets the integer value in the property map
func (pv *PropertyValues) SetValueInt(name string, newValue int) {
	valueString := strconv.FormatInt(int64(newValue), 10)
	pv.SetValue(name, valueString)
}

// NewPropertyValues creates a new set of maps for storing and tracking changes to property values
func NewPropertyValues() *PropertyValues {
	pv := PropertyValues{
		latest:  make(map[string]any),
		changed: make(map[string]any),
	}
	return &pv
}
