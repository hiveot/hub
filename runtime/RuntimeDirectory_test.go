package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils/tlsclient"
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
	var dtThing1ID = tdd.MakeDigiTwinThingID(agentID, agThing1ID)
	var evCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag1, _ := ts.AddConnectAgent(agentID)
	ag1.SetRequestHandler(func(msg *transports.ThingMessage) (stat transports.RequestStatus) {
		stat.Status = vocab.RequestCompleted
		return
	})
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	cl1.SetMessageHandler(func(msg *transports.ThingMessage) {
		// expect 2 events, updated and removed
		evCount.Add(1)
	})
	defer cl1.Disconnect()
	f1 := ts.GetForm(wot.OpSubscribeAllEvents, cl1.GetProtocolType())
	_, err := cl1.SendOperation(f1, "", "", nil, nil, "")
	require.NoError(t, err)

	// Add the TD by sending it as an event
	td1 := tdd.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := json.Marshal(td1)
	//err = ag1.PubTD(agThing1ID, string(td1JSON))
	f2 := ts.GetForm(wot.HTOpUpdateTD, cl1.GetProtocolType())
	// the hub will intercept this operation and update the digitwin directory
	// todo: changes this to follow the directory specification once it is available (assuming it fits)
	_, err = ag1.SendOperation(f2, "", "", td1JSON, nil, "")
	assert.NoError(t, err)

	// Get returns a serialized TD object
	// use the helper directory client rpc method
	td3Json, err := digitwin.DirectoryReadTD(cl1, dtThing1ID)
	require.NoError(t, err)
	var td3 tdd.TD
	err = jsoniter.UnmarshalFromString(td3Json, &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	//stat = cl1.Rpc(nil, directory.ThingID, directory.RemoveTDMethod, &args, nil)
	args4JSON, _ := jsoniter.Marshal(dtThing1ID)
	// RemoveTD from the directory
	f3 := ts.GetForm(wot.OpInvokeAction, cl1.GetProtocolType())
	status, err := cl1.SendOperation(f3, digitwin.DirectoryDThingID, digitwin.DirectoryRemoveTDMethod, string(args4JSON), nil, "")
	require.NoError(t, err)
	require.Equal(t, transports.RequestCompleted, status)

	// after removal of the TD, getTD should return an error but delivery is successful
	status, err = cl1.SendOperation(f3, digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, string(args4JSON), nil, "")
	require.Error(t, err)
	require.Equal(t, vocab.RequestFailed, status)

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
	f1 := ts.GetForm(wot.OpInvokeAction, cl1.GetProtocolType())
	tdList1 := []string{}
	status, err := cl1.SendOperation(f1, digitwin.DirectoryDThingID, digitwin.DirectoryReadAllTDsMethod, args, &tdList1, "")
	require.NoError(t, err)
	assert.Equal(t, transports.RequestCompleted, status)
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

	serverURL := fmt.Sprintf("localhost:%d", ts.Port)
	cl2 := tlsclient.NewTLSClient(serverURL, nil, ts.Certs.CaCert, time.Second*30, "")
	cl2.SetAuthToken(token)

	tdJSONList, err := digitwin.DirectoryReadAllTDs(cl, 100, 0)
	require.NoError(t, err)

	// tds are sent as an array of JSON, first unpack the array of JSON strings
	tdList, err := tdd.UnmarshalTDList(tdJSONList)
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
	cl1.SetMessageHandler(func(msg *transports.ThingMessage) {
		if msg.Operation == vocab.HTOpPublishEvent &&
			msg.ThingID == digitwin.DirectoryDThingID &&
			msg.Name == digitwin.DirectoryEventThingUpdated {

			// decode the TD
			td := tdd.TD{}
			payload := msg.DataAsText()
			err := jsoniter.UnmarshalFromString(payload, &td)
			assert.NoError(t, err)
			tdCount.Add(1)
		}
	})
	//err := cl1.Subscribe("", "")
	f1 := ts.GetForm(wot.OpSubscribeAllEvents, cl1.GetProtocolType())
	_, err := cl1.SendOperation(f1, "", "", nil, nil, "")
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)
	require.NoError(t, err)

	// add a TD
	ts.AddTDs(agentID, 1)
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), tdCount.Load())
}
