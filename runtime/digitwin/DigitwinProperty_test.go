package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateReadProperty(t *testing.T) {
	const agent1ID = "agent1"
	const user1ID = "user1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const propName = "prop1"
	const propValue = 25
	const propValue2 = 52
	const msgID = "msg1"

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event
	tdDoc1 := createTDDoc(thing1ID, 5, 4, 3)
	tdDoc1.AddPropertyAsInt(propName, "", title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agent1ID, string(tdDoc1Json))

	// agent provides a new property value
	err = dtwStore.UpdatePropertyValue(agent1ID, thing1ID, propName, propValue, "")
	assert.NoError(t, err)

	// Read the property value and all values
	dThingID := tdd.MakeDigiTwinThingID(agent1ID, thing1ID)
	v2, err := svc.ReadProperty(user1ID, dThingID, propName)
	assert.NoError(t, err)
	assert.Equal(t, propValue, v2.Data)

	propList, err := svc.ReadAllProperties(user1ID, dThingID)
	assert.Equal(t, 1, len(propList))
	assert.NoError(t, err)

	// next write a new value
	err = dtwStore.UpdatePropertyValue(agent1ID, thing1ID, propName, propValue2, "")
	assert.NoError(t, err)
	v3, err := svc.ReadProperty(user1ID, dThingID, propName)
	assert.Equal(t, propValue2, v3.Data)
}

func TestPropertyReadFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ReadProperty("itsme", "badthingid", "someprop")
	assert.Error(t, err)
	_, err = svc.ReadAllProperties("consumer", "badthingid")
	assert.Error(t, err)
}

func TestPropertyUpdateFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const propName = "prop1"

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1.AddPropertyAsInt(propName, "", "property 1")
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	err = dtwStore.UpdatePropertyValue(agentID, "notathing", propName, 123, "")
	assert.Error(t, err)
	//property names not in the TD are accepted
	err = dtwStore.UpdatePropertyValue(agentID, thingID, "unknownprop", 123, "")
	assert.NoError(t, err)

	//can't write a property that doesn't exist
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	err = dtwStore.WriteProperty("user1", dThingID, "unknownprop", 123, "", "")
	assert.NoError(t, err)
}
