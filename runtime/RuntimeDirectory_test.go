package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
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
	const agentAction = "action1"
	var dtThing1ID = td.MakeDigiTwinThingID(agentID, agThing1ID)
	var evCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag1, _ := ts.AddConnectAgent(agentID)
	ag1.SetRequestHandler(func(req transports.RequestMessage) transports.ResponseMessage {
		// expect 2 requests, updated and removed
		evCount.Add(1)
		return req.CreateResponse(nil, nil)
	})
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	cl1.SetNotificationHandler(func(msg transports.NotificationMessage) {
	})
	defer cl1.Disconnect()
	err := cl1.Subscribe("", "")
	require.NoError(t, err)

	// Add the TD
	td1 := td.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := json.Marshal(td1)
	//err = ag1.PubTD(agThing1ID, string(td1JSON))
	// the hub will intercept this operation and update the digitwin directory
	// todo: changes this to follow the directory specification once it is available (assuming it fits)
	notif := transports.NewNotificationMessage(wot.HTOpUpdateTD, "", "", string(td1JSON))
	err = ag1.SendNotification(notif)
	assert.NoError(t, err)
	time.Sleep(time.Second * 2)
	// Get returns a serialized TD object
	// use the helper directory client rpc method
	time.Sleep(time.Millisecond)
	td3Json, err := digitwin.DirectoryReadTD(cl1, dtThing1ID)
	require.NoError(t, err)
	var td3 td.TD
	err = jsoniter.UnmarshalFromString(td3Json, &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	//send action to agent
	err = cl1.Rpc(wot.OpInvokeAction, dtThing1ID, agentAction, "do something", nil)

	//stat = cl1.SendRequest(nil, directory.ThingID, directory.RemoveTDMethod, &args, nil)
	// RemoveTD from the directory
	err = cl1.Rpc(wot.OpInvokeAction, digitwin.DirectoryDThingID,
		digitwin.DirectoryRemoveTDMethod, dtThing1ID, nil)
	require.NoError(t, err)

	// after removal of the TD, getTD should return an error
	//var tdJSON4 string
	td4Json, err := digitwin.DirectoryReadTD(cl1, dtThing1ID)
	//err = cl1.InvokeAction(digitwin.DirectoryDThingID,
	//	digitwin.DirectoryReadTDMethod, dtThing1ID, &td4Json)
	require.Error(t, err)
	require.Empty(t, td4Json)

	// expect 2 events to be received
	require.Equal(t, int32(1), evCount.Load())
}

func TestReadTDs(t *testing.T) {
	t.Log("--- TestReadTDs start ---")

	const agentID = "agent1"
	const userID = "user1"
	//const agThing1ID = "thing1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 1200)

	//td1JSON, _ := json.Marshal(td1)
	//stat := ag.PubEvent(agThing1ID, vocab.EventNameTD, td1JSON)
	//assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// 1. Use actions
	args := digitwin.DirectoryReadAllTDsArgs{Limit: 10}
	tdList1 := []string{}
	err := cl1.Rpc(wot.OpInvokeAction, digitwin.DirectoryDThingID, digitwin.DirectoryReadAllTDsMethod, args, &tdList1)
	require.NoError(t, err)
	require.True(t, len(tdList1) > 0)

	// 2. Try it the easy way using the generated client code
	tdList2, err := digitwin.DirectoryReadAllTDs(cl1, 333, 02)
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
	cl, token := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 100)

	serverURL := ts.GetServerURL(authn.ClientTypeConsumer)
	cl2 := tlsclient.NewTLSClient(serverURL, nil, ts.Certs.CaCert, time.Second*30)
	cl2.SetAuthToken(token)

	tdJSONList, err := digitwin.DirectoryReadAllTDs(cl, 100, 0)
	require.NoError(t, err)

	// tds are sent as an array of JSON, first unpack the array of JSON strings
	tdList, err := td.UnmarshalTDList(tdJSONList)
	require.NoError(t, err)
	require.Equal(t, 100, len(tdList)) // 100 is the given limit

	// check reading a single td
	tdJSON, err := digitwin.DirectoryReadTD(cl, tdList[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, tdJSON)
}

func TestTDEvent(t *testing.T) {
	t.Log("--- TestTDEvent tests receiving TD update events ---")

	const agentID = "agent1"
	const userID = "user1"
	tdCount := atomic.Int32{}

	r := startRuntime()
	defer r.Stop()
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, token := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	_ = token
	defer cl1.Disconnect()

	// wait to directory TD updated events
	notifHandler := func(msg transports.NotificationMessage) {
		if msg.Operation == vocab.HTOpPublishEvent &&
			msg.ThingID == digitwin.DirectoryDThingID &&
			msg.Name == digitwin.DirectoryEventThingUpdated {

			// decode the TD
			td := td.TD{}
			payload := msg.ToString()
			err := jsoniter.UnmarshalFromString(payload, &td)
			assert.NoError(t, err)
			tdCount.Add(1)
		}
	}
	cl1.SetNotificationHandler(notifHandler)
	//err := cl1.Subscribe("", "")
	err := cl1.Subscribe("", "")
	require.NoError(t, err)
	defer cl1.Unsubscribe("", "")

	time.Sleep(time.Millisecond * 100)
	require.NoError(t, err)

	// add a TD
	ts.AddTDs(agentID, 1)
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), tdCount.Load())
}
