package things

import (
	"strconv"
	"sync"
)

// PropertyValues is a helper type for managing property values and their changes
// This is concurrent safe.
type PropertyValues struct {
	latest  map[string]string
	changed map[string]string
	mux     sync.RWMutex
}

// GetValue returns the string representation of the latest value for the given key
// If the key doesn't exist this returns an empty string
func (pv *PropertyValues) GetValue(key string) (value string, found bool) {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	value, found = pv.latest[key]
	return value, found
}

// GetValues returns the latest or changed values and reset the changed values
// If onlyChanges is set then only return the changes
func (pv *PropertyValues) GetValues(onlyChanges bool) (v map[string]string) {
	if onlyChanges {
		v = pv.GetChanged(true)
	} else {
		v = pv.GetLatest()
	}
	pv.changed = make(map[string]string)
	return v
}

// GetLatest returns the latest map of properties
// This does not reset the 'changed' values so it can be called as often as needed.
func (pv *PropertyValues) GetLatest() map[string]string {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	latestValues := make(map[string]string)
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
func (pv *PropertyValues) GetChanged(clear bool) map[string]string {
	pv.mux.RLock()
	defer pv.mux.RUnlock()
	changedValues := pv.changed
	if clear {
		pv.changed = make(map[string]string)
	} else {
		changedValues = make(map[string]string)
		for k, v := range pv.changed {
			changedValues[k] = v
		}
	}
	return changedValues
}

// SetValue update the properties with a string value
// Returns true if changed.
func (pv *PropertyValues) SetValue(key string, newValue string) {
	pv.mux.Lock()
	defer pv.mux.Unlock()
	oldValue, found := pv.latest[key]
	if !found || oldValue != newValue {
		pv.latest[key] = newValue
		pv.changed[key] = newValue
	}
}

// SetValueBool sets the boolean true/false value in the property map
func (pv *PropertyValues) SetValueBool(key string, newValue bool) {
	valueString := "false"
	if newValue {
		valueString = "true"
	}
	pv.SetValue(key, valueString)
}

// SetValueInt sets the integer value in the property map
func (pv *PropertyValues) SetValueInt(key string, newValue int) {
	valueString := strconv.FormatInt(int64(newValue), 10)
	pv.SetValue(key, valueString)
}

// NewPropertyValues creates a new set of maps for storing and tracking changes to property values
func NewPropertyValues() *PropertyValues {
	pv := PropertyValues{
		latest:  make(map[string]string),
		changed: make(map[string]string),
	}
	return &pv
}
