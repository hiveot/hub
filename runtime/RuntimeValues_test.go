package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryActions(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()
	// agent receives actions and sends events
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer sends actions and receives events
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// step 1: agent publishes a TD: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	key1 := "action-0" // must match TD
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	ag1.SetRequestHandler(func(msg *transports.ThingMessage) (stat transports.RequestStatus) {
		stat.Completed(msg, data, nil)
		return stat
	})
	//err := ag1.PubTD(td1.ID, string(td1JSON))
	f1 := ts.GetForm(wot.HTOpUpdateTD, ag1.GetProtocolType())
	_, err := ag1.SendOperation(f1, td1.ID, "", td1JSON, nil, "")
	require.NoError(t, err)

	// step 2: consumer publish an action to the agent
	cl1.SetMessageHandler(func(msg *transports.ThingMessage) {
	})
	f2 := ts.GetForm(wot.OpInvokeAction, ag1.GetProtocolType())
	status, err := ag1.SendOperation(f2, dThing1ID, key1, data, nil, "")
	//stat := cl1.InvokeAction(dThing1ID, key1, data, nil, "")
	require.NoError(t, err)
	require.NotEqual(t, transports.RequestFailed, status)

	// step 3: read the latest actions from the digital twin
	// first gets its TD
	var tdJSON = ""
	var td tdd.TD

	stat2 := cl1.InvokeAction(digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, dThing1ID, nil, "")
	err, _ = stat2.Decode(&tdJSON)
	err = jsoniter.UnmarshalFromString(tdJSON, &td)
	require.NoError(t, err)

	// get the latest action values from the thing
	// use the API generated from the digitwin TD document using tdd2api
	valueList, err := digitwin.ValuesQueryAllActions(cl1, dThing1ID)
	require.NoError(t, err)
	valueMap := api.ActionListToMap(valueList)

	// value must match that of the action in step 1 and match its requestID
	actVal := valueMap[key1]
	assert.Equal(t, data, actVal.Input)
	assert.Equal(t, stat.CorrelationID, actVal.RequestID)
}

// Get events from the outbox using the experimental http REST api
func TestReadEvents(t *testing.T) {
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
	hc1, token := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer hc1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag1.PubTD(td1.ID, string(td1JSON))
	require.NoError(t, err)

	err = ag1.PubEvent(td1.ID, key1, data, "")
	require.NoError(t, err)

	dtwValues := make([]digitwin.ThingValue, 0)
	stat := hc1.ReadAllEvents(dThing1ID, &dtwValues, "")
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
	cl2, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	var dThingID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag1.PubTD(td1.ID, string(td1JSON))
	require.NoError(t, err)

	err = ag1.PubProperty(td1.ID, key1, data1)
	err = ag1.PubProperty(td1.ID, key2, data2)
	require.NoError(t, err)
	//
	time.Sleep(time.Millisecond)
	valueList, err := digitwin.ValuesReadAllProperties(cl2, dThingID)
	require.NoError(t, err)
	require.Equal(t, 2, len(valueList))
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
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	//var dThingID = tdd.MakeDigiTwinThingID(agentID, td1.ID)
	err := ag1.PubTD(td1.ID, string(td1JSON))

	// consumer subscribes to events/properties
	err = cl1.Subscribe("", "")
	require.NoError(t, err)
	cl1.SetMessageHandler(func(msg *transports.ThingMessage) {
		msgCount.Add(1)
	})
	time.Sleep(time.Millisecond * 100)

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2
	// FIXME: this is agent->consumer
	// consumer SSE client should not send a delivery confirmation!
	err = ag1.PubProperty(td1.ID, key1, data1)
	err = ag1.PubProperty(td1.ID, key2, data2)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(2), msgCount.Load())
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
	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	err := ag1.PubTD(td1.ID, string(td1JSON))

	// agents listen for property write requests
	ag1.SetRequestHandler(func(msg *transports.ThingMessage) (stat transports.RequestStatus) {
		if msg.Operation == vocab.OpWriteProperty && msg.Name == key1 {
			stat.Completed(msg, nil, nil)
			msgCount.Add(1)
		}
		return stat
	})

	// consumer subscribes to events/properties changes
	err = cl1.Observe("", "")
	require.NoError(t, err)
	cl1.SetMessageHandler(func(msg *transports.ThingMessage) {
		// expect an action status message that is the result of invokeaction
		if msg.Name == key1 {
			msgCount.Add(1)
		}
	})
	time.Sleep(time.Millisecond * 100)

	dThingID := tdd.MakeDigiTwinThingID(agentID, td1.ID)
	stat2 := cl1.WriteProperty(dThingID, key1, data1)
	require.Empty(t, stat2.Error)
	require.Equal(t, vocab.RequestDelivered, stat2.Status)
	time.Sleep(time.Millisecond * 100)

	// write property results in an action status message
	// intended to be able to send a failure notice for the request.
	assert.Equal(t, int32(2), msgCount.Load())
	err = cl1.Unobserve("", "")
	assert.NoError(t, err)
}
