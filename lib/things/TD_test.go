package things_test

import (
	vocab2 "github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateTD(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
	assert.NotNil(t, tdoc)

	// Set version
	//versions := map[string]string{"Software": "v10.1", "Hardware": "v2.0"}
	propAffordance := &things.PropertyAffordance{
		DataSchema: things.DataSchema{
			Type:  vocab2.WoTDataTypeArray,
			Title: "version",
		},
	}
	tdoc.UpdateProperty(vocab2.PropDeviceSoftwareVersion, propAffordance)

	// Define TD property
	propAffordance = &things.PropertyAffordance{
		DataSchema: things.DataSchema{
			Type: vocab2.WoTDataTypeString,
			Enum: make([]interface{}, 0), //{"value1", "value2"},
			Unit: "C",
		},
	}

	// created time must be set to RFC3339
	assert.NotEmpty(t, tdoc.Created)
	t1, err := time.Parse(time.RFC3339, tdoc.Created)
	assert.NoError(t, err)
	assert.NotNil(t, t1)

	tdoc.UpdateProperty("prop1", propAffordance)
	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)

	tdoc.UpdateTitleDescription("title", "description")

	tdoc.UpdateAction("action1", &things.ActionAffordance{})
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)

	tdoc.UpdateEvent("event1", &things.EventAffordance{})
	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)

	tdoc.UpdateForms([]things.Form{})

	tid2 := tdoc.GetID()
	assert.Equal(t, thingID, tid2)

	asMap := tdoc.AsMap()
	assert.NotNil(t, asMap)
}

func TestMissingAffordance(t *testing.T) {
	thingID := "urn:thing1"

	// test return nil if no affordance is found
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
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
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
	tdoc.AddProperty("prop1", "", "test property", vocab2.WoTDataTypeBool)

	go func() {
		tdoc.AddProperty("prop2", "", "test property2", vocab2.WoTDataTypeString)
	}()

	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)
	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)
}

// test that IDs with spaces are escaped
func TestAddPropBadIDs(t *testing.T) {
	thingID := "urn:thing 1"
	propID := "prop 1"
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
	tdoc.AddProperty(propID, "", "test property", vocab2.WoTDataTypeBool)

	go func() {
		tdoc.AddProperty("prop2", "", "test property2", vocab2.WoTDataTypeString)
	}()

	prop := tdoc.GetProperty(propID)
	assert.Nil(t, prop)
	prop = tdoc.GetProperty(strings.ReplaceAll(propID, " ", "-"))
	assert.NotNil(t, prop)

	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)
}

func TestAddEvent(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
	tdoc.AddEvent("event1", "", "Test Event", "", nil)

	go func() {
		tdoc.AddEvent("event2", "", "Test Event", "", nil)
	}()

	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)
	time.Sleep(time.Millisecond)
	ev = tdoc.GetEvent("event2")
	assert.NotNil(t, ev)
}

func TestAddAction(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := things.NewTD(thingID, "test TD", vocab2.ThingSensor)
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
