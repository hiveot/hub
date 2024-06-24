package things

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	enc1, err := ser.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := DataSchema{}
	err = ser.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	os := DataSchema{
		Type:       vocab.WoTDataTypeObject,
		Properties: make(map[string]*DataSchema),
	}
	os.Properties["stringProp"] = &DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = &DataSchema{
		Type:    vocab.WoTDataTypeInteger,
		Minimum: 10,
		Maximum: 20,
	}
	enc1, err := ser.Marshal(os)
	assert.NoError(t, err)
	//
	var ds map[string]interface{}
	err = ser.Unmarshal(enc1, &ds)
	assert.NoError(t, err)

	var as DataSchema
	err = ser.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	assert.Equal(t, 10, int(as.Properties["intProp"].Minimum))

}
