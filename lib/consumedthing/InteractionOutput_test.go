package consumedthing

import (
	"log/slog"
	"testing"

	"github.com/hiveot/gocore/logging"
	"github.com/hiveot/gocore/messaging"
	"github.com/hiveot/gocore/wot"
	"github.com/hiveot/gocore/wot/td"

	"github.com/stretchr/testify/assert"
)

const key1ID = "key1"

func MakeTD() *td.TD {
	tdi := td.NewTD("thingID", "test Thing", "devicetype")
	tdi.AddProperty(key1ID, "property 1", "test property", wot.DataTypeString)
	return tdi
}

func TestNilSchema(t *testing.T) {
	logging.SetLogging("", "")
	slog.Info("--- TestNilSchema ---")
	data1 := "text"

	notif := &messaging.NotificationMessage{Name: key1ID, Value: data1}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, notif)

	asValue := io.Value.Text()
	assert.Equal(t, data1, asValue)

}

func TestArray(t *testing.T) {
	data1 := []string{"item 1", "item 2"}
	tv := &messaging.NotificationMessage{Name: key1ID, Value: data1}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, tv)
	asArray := io.Value.Array()
	assert.Len(t, asArray, 2)
}

func TestBool(t *testing.T) {

	tv := &messaging.NotificationMessage{Name: key1ID, Value: true}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, tv)
	asBool := io.Value.Boolean()
	assert.Equal(t, true, asBool)
	asString := io.Value.Text()
	assert.Equal(t, "true", asString)
	asInt := io.Value.Integer()
	assert.Equal(t, 1, asInt)
}

func TestInt(t *testing.T) {
	data1 := 42
	tv := &messaging.NotificationMessage{Name: key1ID, Value: data1}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, tv)
	asInt := io.Value.Integer()
	assert.Equal(t, 42, asInt)
	asString := io.Value.Text()
	assert.Equal(t, "42", asString)
}

func TestString(t *testing.T) {
	data1 := "Hello world"
	tv := &messaging.NotificationMessage{Name: key1ID, Value: data1}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, tv)
	asString := io.Value.Text()
	assert.Equal(t, data1, asString)
}

func TestObject(t *testing.T) {
	//schema := tdd.DataSchema{Type: vocab.DataTypeObject}
	type User struct {
		Name        string
		Age         int
		Active      bool
		LastLoginAt string
	}
	data1 := User{Name: "Bob", Age: 10, Active: true, LastLoginAt: "today"}
	notif := &messaging.NotificationMessage{Name: key1ID, Value: data1}
	tdi := MakeTD()
	ct := NewConsumedThing(tdi, nil)
	io := NewInteractionOutputFromNotification(ct, messaging.AffordanceTypeProperty, notif)
	asMap := io.Value.Map()
	assert.Equal(t, data1.Name, asMap["Name"])
}
