package consumedthing

import (
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

const thing1ID = "thing1"
const key1ID = "key1"

func MakeTD() *td.TD {
	tdi := td.NewTD("thingID", "test Thing", "devicetype")
	tdi.AddProperty(key1ID, "property 1", "test property", wot.WoTDataTypeString)
	return tdi
}

func TestNilSchema(t *testing.T) {
	logging.SetLogging("", "")
	slog.Info("--- TestNilSchema ---")
	data1 := "text"

	notif := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, notif)

	asValue := io.Value.Text()
	assert.Equal(t, data1, asValue)

}

func TestArray(t *testing.T) {
	data1 := []string{"item 1", "item 2"}
	tv := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, tv)
	asArray := io.Value.Array()
	assert.Len(t, asArray, 2)
}

func TestBool(t *testing.T) {
	data1 := true
	tv := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, tv)
	asBool := io.Value.Boolean()
	assert.Equal(t, true, asBool)
	asString := io.Value.Text()
	assert.Equal(t, "true", asString)
	asInt := io.Value.Integer()
	assert.Equal(t, 1, asInt)
}

func TestInt(t *testing.T) {
	data1 := 42
	tv := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, tv)
	asInt := io.Value.Integer()
	assert.Equal(t, 42, asInt)
	asString := io.Value.Text()
	assert.Equal(t, "42", asString)
}

func TestString(t *testing.T) {
	data1 := "Hello world"
	tv := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, tv)
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
	tv := &transports.ResponseMessage{Name: key1ID, Output: data1}
	tdi := MakeTD()
	io := NewInteractionOutputFromResponse(tdi, AffordanceTypeProperty, tv)
	asMap := io.Value.Map()
	assert.Equal(t, data1.Name, asMap["Name"])
}
