package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddRemoveTD(t *testing.T) {
	const agentID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, thing1ID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// use the native thingID for this TD doc as the directory converts it to
	// the digital twin ID using the given agent that owns the TD.
	tdDoc1 := createTDDoc(thing1ID, 5, 4, 3)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.UpdateTD(agentID, thing1ID, string(tddjson))
	require.NoError(t, err)

	dThingID := tdd.MakeDigiTwinThingID(agentID, thing1ID)
	td1b, err := svc.ReadThing(consumerID, dThingID)
	require.NoError(t, err)
	assert.Equal(t, dThing1ID, td1b.ID)

	// after removal, getTD should return nil
	err = svc.RemoveThing("senderID", dThing1ID)
	assert.NoError(t, err)

	td1c, err := svc.ReadThing(consumerID, dThingID)
	assert.Error(t, err)
	assert.Empty(t, td1c)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// bad clientID
	td1, err := svc.ReadThing("", "badid")
	require.Error(t, err)
	require.Empty(t, td1)

	_ = svc
	defer stopFunc()
	tdList, err := svc.ReadAllThings("", 0, 10)
	require.NoError(t, err)
	require.Empty(t, tdList)

}

//func TestQueryTDs(t *testing.T) {
//	_ = os.Remove(testStoreFile)
//	const senderID = "agent1"
//	const thing1ID = "agent1:thing1"
//	const title1 = "title1"
//
//	svc, stopFunc := startDirectory()
//	defer stopFunc()
//
//	tdDoc1 := createTDDoc(thing1ID, title1)
//	err := svc.UpdateTD(senderID, thing1ID, tdDoc1)
//	require.NoError(t, err)
//
//	jsonPathQuery := `$[?(@.id=="agent1:thing1")]`
//	tdList, err := svc.QueryTDs(jsonPathQuery)
//	require.NoError(t, err)
//	assert.NotNil(t, tdList)
//	assert.True(t, len(tdList) > 0)
//	el0 := things.TD{}
//	json.Decode([]byte(tdList[0]), &el0)
//	assert.Equal(t, thing1ID, el0.ID)
//	assert.Equal(t, title1, el0.Title)
//}
