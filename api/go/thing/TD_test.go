package thing_test

import (
	thing2 "github.com/hiveot/hub/api/go/thing"
	vocab2 "github.com/hiveot/hub/api/go/vocab"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hiveot/hub/api/go/vocab"
)

func TestCreateTD(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := thing2.NewTD(thingID, "test TD", vocab.DeviceTypeSensor)
	assert.NotNil(t, tdoc)

	// Set version
	//versions := map[string]string{"Software": "v10.1", "Hardware": "v2.0"}
	propAffordance := &thing2.PropertyAffordance{
		DataSchema: thing2.DataSchema{
			Type:  vocab2.WoTDataTypeArray,
			Title: "version",
		},
	}
	tdoc.UpdateProperty(vocab.VocabSoftwareVersion, propAffordance)

	// Define TD property
	propAffordance = &thing2.PropertyAffordance{
		DataSchema: thing2.DataSchema{
			Type: vocab2.WoTDataTypeString,
			Enum: make([]interface{}, 0), //{"value1", "value2"},
			Unit: "C",
		},
	}

	// created time must be set to ISO8601
	assert.NotEmpty(t, tdoc.Created)
	t1, err := time.Parse(vocab2.ISO8601Format, tdoc.Created)
	assert.NoError(t, err)
	assert.NotNil(t, t1)

	tdoc.UpdateProperty("prop1", propAffordance)
	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)

	tdoc.UpdateTitleDescription("title", "description")

	tdoc.UpdateAction("action1", &thing2.ActionAffordance{})
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)

	tdoc.UpdateEvent("event1", &thing2.EventAffordance{})
	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)

	tdoc.UpdateForms([]thing2.Form{})

	tid2 := tdoc.GetID()
	assert.Equal(t, thingID, tid2)

	asMap := tdoc.AsMap()
	assert.NotNil(t, asMap)
}

func TestMissingAffordance(t *testing.T) {
	thingID := "urn:thing1"

	// test return nil if no affordance is found
	tdoc := thing2.NewTD(thingID, "test TD", vocab2.DeviceTypeSensor)
	assert.NotNil(t, tdoc)

	prop := tdoc.GetProperty("prop1")
	assert.Nil(t, prop)

	action := tdoc.GetAction("action1")
	assert.Nil(t, action)

	ev := tdoc.GetEvent("event1")
	assert.Nil(t, ev)
}

func TestAddProp(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := thing2.NewTD(thingID, "test TD", vocab2.DeviceTypeSensor)
	tdoc.AddProperty("prop1", vocab.VocabUnknown, "test property", vocab2.WoTDataTypeBool, "")

	go func() {
		tdoc.AddProperty("prop2", vocab.VocabUnknown, "test property2", vocab2.WoTDataTypeString, "")
	}()

	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)
	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)
}

func TestAddEvent(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := thing2.NewTD(thingID, "test TD", vocab2.DeviceTypeSensor)
	tdoc.AddEvent("event1", vocab.VocabUnknown, "Test Event", "", nil)

	go func() {
		tdoc.AddEvent("event2", vocab.VocabUnknown, "Test Event", "", nil)
	}()

	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)
	time.Sleep(time.Millisecond)
	ev = tdoc.GetEvent("event2")
	assert.NotNil(t, ev)
}

func TestAddAction(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := thing2.NewTD(thingID, "test TD", vocab2.DeviceTypeSensor)
	tdoc.AddAction("action1", "test", "Test Action", "", nil)

	go func() {
		tdoc.AddAction("action2", "test", "test Action", "", nil)
	}()

	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)
	time.Sleep(time.Millisecond)
	action = tdoc.GetAction("action2")
	assert.NotNil(t, action)
}
