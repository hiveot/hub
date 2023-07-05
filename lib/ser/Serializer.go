package ser

import "encoding/json"

// This package provides the default serializer used by hiveot.
// Rather than hard-coding json everywhere, this allows for easy comparing and changing of serializers.

func JsonMarshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}
func JsonUnmarshal(data []byte, obj interface{}) error {
	return json.Unmarshal(data, obj)
}

// Marshal invokes the default serializer
var Marshal = func(obj interface{}) ([]byte, error) {
	return JsonMarshal(obj)
}

// Unmarshal invokes the defaul deserializer
var Unmarshal = func(data []byte, obj interface{}) error {
	return JsonUnmarshal(data, obj)
}
