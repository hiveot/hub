package td_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestCreateTD(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	assert.NotNil(t, tdoc)

	// Set version
	//versions := map[string]string{"Software": "v10.1", "Hardware": "v2.0"}
	propAffordance := &td.PropertyAffordance{
		DataSchema: td.DataSchema{
			Type:  vocab.WoTDataTypeArray,
			Title: "version",
		},
	}
	tdoc.UpdateProperty(vocab.PropDeviceSoftwareVersion, propAffordance)

	// Define TD property
	propAffordance = &td.PropertyAffordance{
		DataSchema: td.DataSchema{
			Type: vocab.WoTDataTypeString,
			Enum: make([]interface{}, 0), //{"value1", "value2"},
			Unit: "C",
		},
	}
	//propAffordance.SetOneOfValues([]string{})

	thingType := tdoc.AtType
	assert.Equal(t, vocab.ThingSensor, thingType)

	// created time must be set to RFC3339
	assert.NotEmpty(t, tdoc.Created)
	t1, err := time.Parse(time.RFC3339, tdoc.Created)
	assert.NoError(t, err)
	assert.NotNil(t, t1)

	tdoc.UpdateProperty("prop1", propAffordance)
	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)

	tdoc.UpdateTitleDescription("title", "description")

	tdoc.UpdateAction("action1", &td.ActionAffordance{})
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)

	tdoc.UpdateEvent("event1", &td.EventAffordance{})
	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)

	tdoc.SetForms([]td.Form{})

	tid2 := tdoc.GetID()
	assert.Equal(t, thingID, tid2)

	asMap := tdoc.AsMap()
	assert.NotNil(t, asMap)
}

func TestMissingAffordance(t *testing.T) {
	thingID := "urn:thing1"

	// test return nil if no affordance is found
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
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
	prop2AtType := "test:specialtype"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	tdoc.AddProperty("prop1", "prop 1", "test property", vocab.WoTDataTypeBool)

	aff := tdoc.AddProperty("prop2", "test property2", "", vocab.WoTDataTypeString)
	aff.SetAtType(prop2AtType)
	aff.Unit = vocab.UnitPercent

	// retrieve a property by its @type value
	pa2Name, pa2 := tdoc.GetPropertyOfVocabType(prop2AtType)
	assert.Equal(t, "prop2", pa2Name)
	assert.NotNil(t, pa2)

	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)
	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)

	tdoc.AddPropertyAsBool("b1", "bool1", "boolean 1")
	tdoc.AddPropertyAsString("s1", "string1", "string 1")
	tdoc.AddPropertyAsInt("i1", "int1", "integer 1")

}

// test that IDs with spaces are escaped
func TestAddPropBadIDs(t *testing.T) {
	thingID := "urn:thing 1"
	propID := "prop 1"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	tdoc.AddProperty(propID, "test property", "", vocab.WoTDataTypeBool)

	tdoc.AddProperty("prop2", "test property2", "", vocab.WoTDataTypeString)

	prop := tdoc.GetProperty(propID)
	assert.Nil(t, prop)
	prop = tdoc.GetProperty(strings.ReplaceAll(propID, " ", "_"))
	assert.NotNil(t, prop)

	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)
}

func TestAddEvent(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	tdoc.AddEvent("event1", "Test Event", "", nil)

	tdoc.AddEvent("event2", "Test Event", "", nil)

	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)
	time.Sleep(time.Millisecond)
	ev = tdoc.GetEvent("event2")
	assert.NotNil(t, ev)
}

func TestAddAction(t *testing.T) {
	thingID := "urn:thing1"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	tdoc.AddAction("action1", "test", "Test Action", nil)

	// has a space
	tdoc.AddAction("action 2", "test", "test Action", nil)
	tdoc.EscapeKeys()
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)
	time.Sleep(time.Millisecond)
	action = tdoc.GetAction("action_2")
	assert.NotNil(t, action)
}

// just some basic tests. need much more
func TestForms(t *testing.T) {
	const action1Name = "action1"
	const prop1Name = "prop1"
	const event1Name = "event1"
	thingID := "urn:thing1"
	tdoc := td.NewTD(thingID, "test TD", vocab.ThingSensor)
	actAff := tdoc.AddAction(action1Name, "action", "Test Action", nil)
	tdoc.AddProperty(prop1Name, "prop", "Test Prop", wot.DataTypeInteger)
	tdoc.AddEvent(event1Name, "event", "Test Event", nil)

	actForm := td.NewForm(wot.OpInvokeAction, "https://localhost/action")
	actAff.Forms = []td.Form{actForm}

	forms := make([]td.Form, 0)
	//forms = append(forms, td.NewForm(wot.OpInvokeAction, "https://localhost/action"))
	forms = append(forms, td.NewForm(wot.OpWriteProperty, "https://localhost/prop"))
	forms = append(forms, td.NewForm(wot.OpSubscribeEvent, "https://localhost/ev"))
	tdoc.AddForms(forms)

	//
	f1 := tdoc.GetForms(wot.OpWriteProperty, prop1Name)
	require.NotNil(t, f1)
	f2 := tdoc.GetForms(wot.OpInvokeAction, action1Name)
	require.NotNil(t, f2)
	f3 := tdoc.GetForms(wot.OpSubscribeEvent, event1Name)
	require.NotNil(t, f3)

	uriVars := make(map[string]string)
	f1href, err := tdoc.GetFormHRef(f1[0], uriVars)
	assert.NoError(t, err)
	assert.NotEmpty(t, f1href)
}
