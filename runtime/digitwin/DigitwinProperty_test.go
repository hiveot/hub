package digitwin_test

import (
	"encoding/json"
	"testing"

	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateReadProperty(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agent1ID = "agent1"
	const user1ID = "user1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const propName = "prop1"
	const propValue = 25
	const propValue2 = 52
	const correlationID = "request-1"
	dThing1ID := td.MakeDigiTwinThingID(agent1ID, thing1ID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event
	tdDoc1 := createTDDoc(thing1ID, 5, 4, 3)
	tdDoc1.AddPropertyAsInt(propName, "", title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agent1ID, string(tdDoc1Json))

	//

	// agent has provided a new property value
	changed, err := dtwStore.UpdatePropertyValue(
		digitwin.ThingValue{ThingID: dThing1ID, Name: propName, Data: propValue})
	assert.NoError(t, err)
	assert.True(t, changed)

	// Read the property value and all values
	v2, err := svc.ValuesSvc.ReadProperty(user1ID, digitwin.ThingValuesReadPropertyArgs{
		ThingID: dThing1ID,
		Name:    propName})
	assert.NoError(t, err)
	assert.Equal(t, propValue, v2.Data)

	propList, err := svc.ValuesSvc.ReadAllProperties(user1ID, dThing1ID)
	assert.Equal(t, 1, len(propList))
	assert.NoError(t, err)

	// next write a new value
	changed, err = dtwStore.UpdatePropertyValue(
		digitwin.ThingValue{ThingID: dThing1ID, Name: propName, Data: propValue2})

	assert.NoError(t, err)
	assert.True(t, changed)
	v3, err := svc.ValuesSvc.ReadProperty(user1ID, digitwin.ThingValuesReadPropertyArgs{
		ThingID: dThing1ID,
		Name:    propName})
	assert.Equal(t, propValue2, v3.Data)
}

func TestPropertyReadFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.ReadProperty("itsme", digitwin.ThingValuesReadPropertyArgs{
		ThingID: "badthingid",
		Name:    "someprop"})
	assert.Error(t, err)
	_, err = svc.ValuesSvc.ReadAllProperties("consumer", "badthingid")
	assert.Error(t, err)
}

func TestPropertyUpdateFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	const propName = "prop1"
	dThing1ID := td.MakeDigiTwinThingID(agentID, thingID)
	dBadThingID := td.MakeDigiTwinThingID(agentID, "notathing")

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1.AddPropertyAsInt(propName, "", "property 1")
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	changed, err := dtwStore.UpdatePropertyValue(
		digitwin.ThingValue{ThingID: dBadThingID, Name: propName, Data: 123})
	assert.Error(t, err)
	assert.False(t, changed)
	//property names not in the TD are accepted
	changed, err = dtwStore.UpdatePropertyValue(
		digitwin.ThingValue{ThingID: dThing1ID, Name: "unknownprop", Data: 123})
	assert.NoError(t, err)
	assert.True(t, changed)

	//can't update a property that doesn't exist
	//dThingID := td.MakeDigiTwinThingID(agentID, thingID)
	//err = dtwStore.WriteProperty(dThingID, digitwin.ThingValue{
	//	Name:     "unknownprop",
	//	Data:     123,
	//	SenderID: "user1",
	//})
	//assert.NoError(t, err)
}
