package consumedthing

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

const thing1ID = "thing1"
const key1ID = "key1"

func TestNilSchema(t *testing.T) {
	logging.SetLogging("", "")
	slog.Info("--- TestNilSchema ---")
	data1 := "text"

	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)

	asValue := io.Value.Text()
	assert.Equal(t, data1, asValue)

}

func TestArray(t *testing.T) {
	data1 := []string{"item 1", "item 2"}
	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)
	asArray := io.Value.Array()
	assert.Len(t, asArray, 2)
}

func TestBool(t *testing.T) {
	data1 := true
	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)
	asBool := io.Value.Boolean()
	assert.Equal(t, true, asBool)
	asString := io.Value.Text()
	assert.Equal(t, "true", asString)
	asInt := io.Value.Integer()
	assert.Equal(t, 1, asInt)
}

func TestInt(t *testing.T) {
	data1 := 42
	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)
	asInt := io.Value.Integer()
	assert.Equal(t, 42, asInt)
	asString := io.Value.Text()
	assert.Equal(t, "42", asString)
}

func TestString(t *testing.T) {
	data1 := "Hello world"
	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)
	asString := io.Value.Text()
	assert.Equal(t, data1, asString)
}

func TestObject(t *testing.T) {
	//schema := tdd.DataSchema{Type: vocab.WoTDataTypeObject}
	type User struct {
		Name        string
		Age         int
		Active      bool
		LastLoginAt string
	}
	data1 := User{Name: "Bob", Age: 10, Active: true, LastLoginAt: "today"}
	tm := hubclient.NewThingMessage(vocab.MessageTypeEvent, thing1ID, key1ID, data1, "")
	io := NewInteractionOutputFromTM(tm, nil)
	asMap := io.Value.Map()
	assert.Equal(t, data1.Name, asMap["Name"])
}