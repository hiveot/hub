package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/httpsse"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestAddRemoveTD(t *testing.T) {
	t.Log("--- TestAddRemoveTD start ---")
	defer t.Log("--- TestAddRemoveTD end ---")

	const agentID = "agent1"
	const userID = "user1"
	const agThing1ID = "thing1"
	var dtThing1ID = tdd.MakeDigiTwinThingID(agentID, agThing1ID)
	var evCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	ag.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		stat.Progress = vocab.ProgressStatusCompleted
		return
	})
	defer ag.Disconnect()
	cl, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	cl.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		// expect 2 events, updated and removed
		evCount.Add(1)
		return stat
	})
	defer cl.Disconnect()
	err := cl.Subscribe("", "")

	// Add the TD by sending it as an event
	td1 := tdd.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := json.Marshal(td1)
	err = ag.PubTD(agThing1ID, string(td1JSON))
	assert.NoError(t, err)

	// Get returns a serialized TD object
	// use the helper directory client rpc method
	td3Json, err := digitwin.DirectoryReadDTD(cl, dtThing1ID)
	require.NoError(t, err)
	var td3 tdd.TD
	err = json.Unmarshal([]byte(td3Json), &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	//stat = cl.Rpc(nil, directory.ThingID, directory.RemoveTDMethod, &args, nil)
	args4JSON, _ := json.Marshal(dtThing1ID)
	stat := cl.InvokeAction(digitwin.DirectoryDThingID, digitwin.DirectoryRemoveDTDMethod, string(args4JSON), "")
	require.Empty(t, stat.Error)

	// after removal of the TD, getTD should return an error but delivery is successful
	stat = cl.InvokeAction(digitwin.DirectoryDThingID, digitwin.DirectoryReadDTDMethod, string(args4JSON), "")
	require.NotEmpty(t, stat.Error)
	require.Equal(t, vocab.ProgressStatusCompleted, stat.Progress)

	// expect 2 events to be received
	require.Equal(t, int32(2), evCount.Load())
}

func TestReadTDs(t *testing.T) {
	t.Log("--- TestReadTDs start ---")

	const agentID = "agent1"
	const userID = "user1"
	//const agThing1ID = "thing1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	cl, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 1200)

	//td1JSON, _ := json.Marshal(td1)
	//stat := ag.PubEvent(agThing1ID, vocab.EventNameTD, td1JSON)
	//assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// 1. Use actions
	args := digitwin.DirectoryReadAllDTDsArgs{Limit: 10}
	stat := cl.InvokeAction(digitwin.DirectoryDThingID, digitwin.DirectoryReadAllDTDsMethod, args, "")
	require.Empty(t, stat.Error)
	assert.NotNil(t, stat.Reply)
	tdList1 := []string{}
	err, hasData := stat.Decode(&tdList1)
	require.NoError(t, err)
	require.True(t, hasData)
	require.True(t, len(tdList1) > 0)

	// 2. Try it the easy way
	tdList2, err := digitwin.DirectoryReadAllDTDs(cl, 333, 02)
	require.NoError(t, err)
	require.True(t, len(tdList2) > 0)
}

func TestReadTDsRest(t *testing.T) {
	t.Log("--- TestReadTDs using the rest api ---")

	const agentID = "agent1"
	const userID = "user1"

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	cl, token := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 100)

	serverURL := fmt.Sprintf("localhost:%d", ts.Port)
	cl2 := tlsclient.NewTLSClient(serverURL, nil, ts.Certs.CaCert, time.Second*30, "")
	cl2.SetAuthToken(token)

	data, _, err := cl2.Get(httpsse.GetAllThingsPath)
	require.NoError(t, err)

	// tds are sent as an array of JSON, first unpack the array of JSON strings
	tdJSONList := []string{}
	err = json.Unmarshal(data, &tdJSONList)
	require.NoError(t, err)
	tdList, err := tdd.UnmarshalTDList(tdJSONList)
	require.NoError(t, err)
	require.Equal(t, 100, len(tdList))

	// read a single td
	vars := map[string]string{"thingID": tdList[0].ID}
	getThingPath := utils.Substitute(httpsse.GetThingPath, vars)
	data, _, err = cl2.Get(getThingPath)
	require.NoError(t, err)
}

func TestTDEvent(t *testing.T) {
	t.Log("--- TestTDEvent tests receiving TD update events ---")

	const agentID = "agent1"
	const userID = "user1"
	tdCount := atomic.Int32{}

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	cl, token := ts.AddConnectUser(userID, authz.ClientRoleManager)
	_ = token
	defer cl.Disconnect()

	// wait to directory TD updated events
	cl.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		stat.Completed(msg, nil, nil)
		if msg.MessageType == vocab.MessageTypeEvent &&
			msg.ThingID == digitwin.DirectoryDThingID &&
			msg.Name == service.ThingUpdatedEventName {

			// decode the TD
			td := tdd.TD{}
			payload := msg.DataAsText()
			err := json.Unmarshal([]byte(payload), &td)
			assert.NoError(t, err)
			tdCount.Add(1)
		}

		return stat
	})
	err := cl.Subscribe("", "")
	time.Sleep(time.Millisecond * 100)
	require.NoError(t, err)

	// add a TD
	ts.AddTDs(agentID, 1)
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), tdCount.Load())
}
