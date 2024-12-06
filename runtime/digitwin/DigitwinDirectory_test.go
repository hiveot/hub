package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddRemoveTD(t *testing.T) {
	const agentID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	var dThing1ID = td.MakeDigiTwinThingID(agentID, thing1ID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()
	dirSvc := svc.DirSvc

	// use the native thingID for this TD doc as the directory converts it to
	// the digital twin ID using the given agent that owns the TD.
	tdd1 := createTDDoc(thing1ID, 5, 4, 3)
	tdd1JSON, _ := json.Marshal(tdd1)
	err := dirSvc.UpdateTD(agentID, string(tdd1JSON))
	require.NoError(t, err)

	dThingID := td.MakeDigiTwinThingID(agentID, thing1ID)
	tdd2JSON, err := dirSvc.ReadTD(consumerID, dThingID)
	require.NoError(t, err)
	require.NotEmpty(t, tdd2JSON)

	dtdList, err := dirSvc.ReadAllTDs(consumerID,
		digitwin.DirectoryReadAllTDsArgs{Limit: 10})
	assert.NoError(t, err)
	assert.NotEmpty(t, dtdList)

	// after removal, getTD should return nil
	err = dirSvc.RemoveTD("senderID", dThing1ID)
	assert.NoError(t, err)

	td1c, err := dirSvc.ReadTD(consumerID, dThingID)
	assert.Error(t, err)
	assert.Empty(t, td1c)

}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"

	svc, _, stopFunc := startService(true)
	defer stopFunc()
	dirSvc := svc.DirSvc

	// bad clientID
	td1, err := dirSvc.ReadTD("", "badid")
	require.Error(t, err)
	require.Empty(t, td1)

	_ = svc
	defer stopFunc()
	tdList, err := svc.ReadAllTDs("", 0, 10)
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
//	err := svc.UpdateDTD(senderID, thing1ID, tdDoc1)
//	require.NoError(t, err)
//
//	jsonPathQuery := `$[?(@.id=="agent1:thing1")]`
//	tdList, err := svc.QueryDTDs(jsonPathQuery)
//	require.NoError(t, err)
//	assert.NotNil(t, tdList)
//	assert.True(t, len(tdList) > 0)
//	el0 := things.TD{}
//	json.Decode([]byte(tdList[0]), &el0)
//	assert.Equal(t, thing1ID, el0.ID)
//	assert.Equal(t, title1, el0.Title)
//}
