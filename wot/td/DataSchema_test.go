package td_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	jsoniter "github.com/json-iterator/go"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := td.DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	enc1, err := jsoniter.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := td.DataSchema{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	os := td.DataSchema{
		Type:       vocab.WoTDataTypeObject,
		Properties: make(map[string]*td.DataSchema),
	}
	os.Properties["stringProp"] = &td.DataSchema{
		Type:            vocab.WoTDataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = &td.DataSchema{
		Type:    vocab.WoTDataTypeInteger,
		Minimum: 10,
		Maximum: 20,
	}
	enc1, err := jsoniter.Marshal(os)
	assert.NoError(t, err)
	//
	var ds map[string]interface{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)

	var as td.DataSchema
	err = jsoniter.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	assert.Equal(t, 10, int(as.Properties["intProp"].Minimum))

}
