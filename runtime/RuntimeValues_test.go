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
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestHttpsGetActions(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()
	// agent receives actions and sends events
	cl1, _ := ts.AddConnectAgent(agentID)
	defer cl1.Disconnect()
	// consumer sends actions and receives events
	cl2, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()

	// step 1: agent publishes a TD: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	key1 := "action-0" // must match TD
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	cl1.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		stat.Completed(msg, data, nil)
		return stat
	})
	err := cl1.PubTD(td1.ID, string(td1JSON))
	require.NoError(t, err)

	// step 2: consumer publish an action to the agent
	cl2.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		return stat
	})
	stat := cl2.InvokeAction(dThing1ID, key1, data, nil, "")
	require.Empty(t, stat.Error)

	// step 3: read the latest actions from the digital twin
	// first gets its TD
	var tdJSON = ""
	var td tdd.TD

	stat2 := cl2.InvokeAction(digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, dThing1ID, nil, "")
	err, _ = stat2.Decode(&tdJSON)
	err = jsoniter.UnmarshalFromString(tdJSON, &td)
	require.NoError(t, err)

	// get the latest action values from the thing
	// use the API generated from the digitwin TD document using tdd2api
	valueList, err := digitwin.ValuesQueryAllActions(cl2, dThing1ID)
	require.NoError(t, err)
	valueMap := api.ActionListToMap(valueList)

	// value must match that of the action in step 1 and match its messageID
	actVal := valueMap[key1]
	assert.Equal(t, data, actVal.Input)
	assert.Equal(t, stat.MessageID, actVal.MessageID)
}

// Get events from the outbox using the experimental http REST api
func TestHttpsGetEvents(t *testing.T) {
	const agentID = "agent1"
	const key1 = "key1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()

	// agent for publishing events
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer for reading events
	hc1, token := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer hc1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag1.PubTD(td1.ID, string(td1JSON))
	require.NoError(t, err)

	err = ag1.PubEvent(td1.ID, key1, data, "")
	require.NoError(t, err)

	// read using a plain old http client
	hostPort := fmt.Sprintf("localhost:%d", ts.Port)
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, ts.Certs.CaCert, time.Minute, "")
	tlsClient.SetAuthToken(token)

	// read latest using the http REST API
	vars := map[string]string{"thingID": dThing1ID}
	eventPath := utils.Substitute(httpsse.ReadAllEventsPath, vars)
	reply, _, err := tlsClient.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, reply)

	dtwValues := make([]digitwin.ThingValue, 0)
	err = jsoniter.Unmarshal(reply, &dtwValues)
	require.NoError(t, err)
	require.NotZero(t, len(dtwValues))

	// read latest using the generated client api
	valueList, err := digitwin.ValuesReadAllEvents(hc1, dThing1ID)
	//resp, err := digitwin.OutboxReadLatest(hc, "", nil, "", dThingID)
	require.NoError(t, err)
	require.NotNil(t, valueList)
	valueMap := api.ValueListToMap(valueList)
	require.Equal(t, len(dtwValues), len(valueMap))
	require.Equal(t, data, valueMap[key1].Data)
}

func TestHttpsGetProps(t *testing.T) {
	const agentID = "agent1"
	const key1 = "key1"
	const key2 = "key2"
	const userID = "user1"
	const data1 = "Hello world"
	const data2 = 25

	r := startRuntime()
	defer r.Stop()

	// agent publishes properties
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer reads properties
	cl2, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	var dThingID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag1.PubTD(td1.ID, string(td1JSON))
	require.NoError(t, err)

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2
	err = ag1.PubProperties(td1.ID, propMap)
	require.NoError(t, err)
	//
	valueList, err := digitwin.ValuesReadAllProperties(cl2, dThingID)
	require.NoError(t, err)
	require.Equal(t, len(propMap), len(valueList))
	valueMap := api.ValueListToMap(valueList)

	// note: golang unmarshalls integers as float64.
	data2raw := valueMap[key2].Data.(float64)
	require.Equal(t, data2, int(data2raw))
}

func TestSubscribeValues(t *testing.T) {
	t.Log("--- TestSubscribeValues tests receiving value update events ---")
	const agentID = "agent1"
	const userID = "user1"
	const key1 = "key1"
	const key2 = "key2"
	const data1 = "Hello world"
	const data2 = 25
	var msgCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	hc, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer hc.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	//var dThingID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag.PubTD(td1.ID, string(td1JSON))

	// consumer subscribes to events/properties
	err = hc.Subscribe("", "")
	require.NoError(t, err)
	hc.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		stat.Completed(msg, nil, nil)
		msgCount.Add(1)
		return stat
	})
	time.Sleep(time.Millisecond * 100)

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2
	// FIXME: this is agent->consumer
	// consumer SSE client should not send a delivery confirmation!
	err = ag.PubProperties(td1.ID, propMap)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), msgCount.Load())
}

func TestWriteProperties(t *testing.T) {
	t.Log("--- TestWriteProperties ---")
	const agentID = "agent1"
	const userID = "user1"
	const key1 = "key1"
	const data1 = "Hello world"
	var msgCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	cl, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	err := ag.PubTD(td1.ID, string(td1JSON))

	// agents listen for property write requests
	ag.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		if msg.MessageType == vocab.MessageTypeProperty && msg.Name == key1 {
			stat.Completed(msg, nil, nil)
			msgCount.Add(1)
		}
		return stat
	})

	// consumer subscribes to events/properties changes
	err = cl.Observe("", "")
	require.NoError(t, err)
	//var tv digitwin.ThingValue
	cl.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.ActionProgress) {
		if msg.Name == key1 {
			stat.Completed(msg, nil, nil)
			msgCount.Add(1)
		}
		return stat
	})
	time.Sleep(time.Millisecond * 100)

	dThingID := tdd.MakeDigiTwinThingID(agentID, td1.ID)
	stat2 := cl.WriteProperty(dThingID, key1, data1)
	require.Empty(t, stat2.Error)
	require.Equal(t, vocab.ProgressStatusCompleted, stat2.Progress)

	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), msgCount.Load())
	err = cl.Unobserve("", "")
	assert.NoError(t, err)
}
