package thing

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	enc1, err := json.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := DataSchema{}
	err = json.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	os := DataSchema{
		Type:       vocab.WoTDataTypeObject,
		Properties: make(map[string]DataSchema),
	}
	os.Properties["stringProp"] = DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = DataSchema{
		Type:          vocab.WoTDataTypeInteger,
		NumberMinimum: 10,
		NumberMaximum: 20,
	}
	enc1, err := json.Marshal(os)
	assert.NoError(t, err)
	//
	var ds map[string]interface{}
	err = json.Unmarshal(enc1, &ds)
	assert.NoError(t, err)

	var as DataSchema
	err = json.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	assert.Equal(t, 10, int(as.Properties["intProp"].NumberMinimum))

}
