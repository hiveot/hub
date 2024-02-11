package things

type ThingValueMap map[string]*ThingValue

// Age returns the age of a property, or "" if it doesn't exist
// intended for use in template as .Values.Age $key
func (vm ThingValueMap) Age(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return tv.Age()
}

// Get returns the value of a property key, or nil if it doesn't exist
func (vm ThingValueMap) Get(key string) *ThingValue {
	tv, found := vm[key]
	if !found {
		return nil
	}
	return tv
}

// ToString returns the value of a property as text, or "" if it doesn't exist
// intended for use in template as .Values.ToString $key
func (vm ThingValueMap) ToString(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return string(tv.Data)
}

// SenderID returns the sender ID of the client updating the value, or "" if it doesn't exist
// intended for use in template as .Values.SenderID $key
func (vm ThingValueMap) SenderID(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return tv.AgentID
}

// Set a property value in the map
// if key already exists its value will be replaced
func (vm ThingValueMap) Set(key string, tv *ThingValue) {
	vm[key] = tv
}

// Updated returns the timestamp of a property, or "" if it doesn't exist
// intended for use in template as .Values.Updated $key
func (vm *ThingValueMap) Updated(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return tv.Updated()
}

func NewThingValueMap() ThingValueMap {
	vm := make(ThingValueMap)
	return vm
}
