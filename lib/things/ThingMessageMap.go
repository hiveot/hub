package things

// ThingMessageMap map of message event or action keys to value
type ThingMessageMap map[string]*ThingMessage

// Age returns the age of a property, or "" if it doesn't exist
// intended for use in template as .Values.Age $key
//func (vm ThingMessageMap) Age(key string) string {
//	tv := vm.Get(key)
//	if tv == nil {
//		return ""
//	}
//	return tv.Age()
//}

// Get returns the value of a property key, or nil if it doesn't exist
func (vm ThingMessageMap) Get(key string) *ThingMessage {
	tv, found := vm[key]
	if !found {
		return nil
	}
	return tv
}

// GetUpdated returns the timestamp of a property, or "" if it doesn't exist
// intended for use in template as .Values.GetUpdated $key
func (vm *ThingMessageMap) GetUpdated(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return tv.GetUpdated()
}

// ToString returns the value of a property as text, or "" if it doesn't exist
// intended for use in template as .Values.ToString $key
func (vm ThingMessageMap) ToString(key string) string {
	tv := vm.Get(key)
	if tv == nil {
		return ""
	}
	return string(tv.Data)
}

// Set a property value in the map
// if key already exists its value will be replaced
func (vm ThingMessageMap) Set(key string, tv *ThingMessage) {
	vm[key] = tv
}

// NewThingMessageMap creates map of message event,action or rpc key to value
func NewThingMessageMap() ThingMessageMap {
	vm := make(ThingMessageMap)
	return vm
}