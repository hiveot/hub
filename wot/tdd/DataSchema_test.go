package tdd_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/wot/tdd"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := tdd.DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	enc1, err := ser.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := tdd.DataSchema{}
	err = ser.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	os := tdd.DataSchema{
		Type:       vocab.WoTDataTypeObject,
		Properties: make(map[string]*tdd.DataSchema),
	}
	os.Properties["stringProp"] = &tdd.DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = &tdd.DataSchema{
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

	var as tdd.DataSchema
	err = ser.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	assert.Equal(t, 10, int(as.Properties["intProp"].Minimum))

}
