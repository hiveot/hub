package runtime_test

import (
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	authz "github.com/hiveot/hub/runtime/authz/api"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryActions(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"
	const actionID = "action-0"
	var updateChan1 = make(chan bool)

	r := startRuntime()
	defer r.Stop()
	// agent receives actions and sends events
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer sends actions and receives events
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// step 1: agent publishes a TD: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	var dThing1ID = td.MakeDigiTwinThingID(agentID, td1.ID)

	ag1.SetRequestHandler(func(msg *messaging.RequestMessage,
		c messaging.IConnection) *messaging.ResponseMessage {

		slog.Info("request: "+msg.Operation, "correlationID", msg.CorrelationID)
		return msg.CreateResponse(data, nil)
	})

	co1.SetNotificationHandler(func(msg *messaging.NotificationMessage) {
		slog.Info("notification: " + msg.Operation)
		// signal notification received
		updateChan1 <- true
	})
	err := co1.Subscribe("", "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	//err = ag1.UpdateThing(td1)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	err = digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)

	// step 2: consumer publish an action to the agent it should return as
	// a notification.
	require.NoError(t, err)
	<-updateChan1

	// this action is recorded
	err = co1.InvokeAction(dThing1ID, actionID, data, nil)
	require.NoError(t, err)

	// get the latest action values from the thing
	// use the API generated from the digitwin TD document using tdd2api
	apiActionStatusMap, err := digitwin.ThingValuesQueryAllActions(co1, dThing1ID)
	require.NoError(t, err)
	// this should be convertable to the messaging.ActionStatus map
	var msgActionStatusMap map[string]messaging.ActionStatus
	err = utils.Decode(apiActionStatusMap, &msgActionStatusMap)
	require.NoError(t, err)

	// value must match that of the action in step 1 and match its correlationID
	actVal := msgActionStatusMap[actionID]
	assert.Equal(t, data, actVal.Input)
	assert.Equal(t, messaging.StatusCompleted, actVal.State)

	// last, use consumer to query actions
	actionMap2, err := co1.QueryAllActions(dThing1ID)
	require.NoError(t, err)
	assert.Equal(t, len(msgActionStatusMap), len(actionMap2))
}

// Get events from the outbox using the experimental http REST api
func TestReadEvents(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const key1 = "key1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()

	// agent for publishing events
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer for reading events
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	// is requested. hiveot uses it to determine if a response is required.
	td1 := ts.CreateTestTD(0)
	//err := ag1.UpdateThing(td1)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	err := digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)

	// step 2: agent publishes an event
	err = ag1.PubEvent(td1.ID, key1, data)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 1)

	// step 3: read all events
	var dThing1ID = td.MakeDigiTwinThingID(agentID, td1.ID)
	dtwValues, err := digitwin.ThingValuesReadAllEvents(co1, dThing1ID)
	require.NoError(t, err)
	require.NotZero(t, len(dtwValues))
	require.Equal(t, data, dtwValues[key1].Data)

	// read latest using the generated client api
	valueMap, err := digitwin.ThingValuesReadAllEvents(co1, dThing1ID)
	require.NoError(t, err)
	require.NotNil(t, valueMap)
	require.Equal(t, len(dtwValues), len(valueMap))
	require.Equal(t, data, valueMap[key1].Data)
}

func TestHttpsGetProps(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const key1 = "key1"
	const key2 = "key2"
	const userID = "user1"
	const data1 = "Hello world"
	const data2 = 25

	r := startRuntime()
	defer r.Stop()

	// agent publishes properties
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	// consumer reads properties
	cl2, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	err := digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)
	time.Sleep(time.Millisecond * 10)
	require.NoError(t, err)

	//err = ag1.PubProperty(td1.ID, key1, data1)
	//err = ag1.PubProperty(td1.ID, key2, data2)
	err = ag1.PubProperty(td1.ID, key1, data1)
	err = ag1.PubProperty(td1.ID, key2, data2)

	require.NoError(t, err)
	//
	time.Sleep(time.Millisecond * 10)
	var dThingID = td.MakeDigiTwinThingID(agentID, td1.ID)
	valueMap, err := cl2.ReadAllProperties(dThingID)
	//valueMap, err := digitwin.ThingValuesReadAllProperties(cl2, dThingID)
	require.NoError(t, err)
	require.Equal(t, 2, len(valueMap))

	// note: golang unmarshalls integers as float64.
	data2raw := valueMap[key2].Data.(float64)
	require.Equal(t, data2, int(data2raw))
}

func TestSubscribeValues(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const key1 = "key1"
	const key2 = "key2"
	const data1 = "Hello world"
	const data2 = 25
	var msgCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// 1. consumer subscribes to events/properties
	err := cl1.Subscribe("", "")
	require.NoError(t, err)
	cl1.SetNotificationHandler(func(msg *messaging.NotificationMessage) {
		msgCount.Add(1)
	})

	// 2: agent creates a TD first: dtw:agent1:thing-1 - this sends a notification
	td1 := ts.CreateTestTD(0)

	// 3: agent publishes notification
	//err = ag1.UpdateThing(td1)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	err = digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)

	time.Sleep(time.Millisecond * 100)

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2

	err = ag1.PubEvent(td1.ID, key1, data1)
	err = ag1.PubEvent(td1.ID, key2, data2)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)
	// one thing updated and two notification events
	assert.Equal(t, int32(3), msgCount.Load())
}

func TestWriteProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const key1 = "key1"
	const data1 = "Hello world"
	var msgCount atomic.Int32

	r := startRuntime()
	defer r.Stop()
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	//err := ag1.UpdateThing(td1)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	err := digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)

	// agents listen for property write requests
	ag1.SetRequestHandler(func(msg *messaging.RequestMessage,
		c messaging.IConnection) *messaging.ResponseMessage {
		if msg.Operation == vocab.OpWriteProperty && msg.Name == key1 {
			msgCount.Add(1)
		}
		return msg.CreateResponse(msg.Input, nil)
	})

	// consumer subscribes to events/properties changes
	err = co1.ObserveProperty("", "")
	require.NoError(t, err)

	co1.SetResponseHandler(func(msg *messaging.ResponseMessage) error {
		// expect an action status message that is the result of invokeaction
		if msg.Name == key1 {
			msgCount.Add(1)
		}
		return nil
	})
	time.Sleep(time.Millisecond * 100)

	dThingID := td.MakeDigiTwinThingID(agentID, td1.ID)
	//stat2 := co1.WriteProperty(dThingID, key1, data1)
	//require.Empty(t, stat2.Error)
	err = co1.Rpc(wot.OpWriteProperty, dThingID, key1, data1, nil)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	// write property results in a request on the agent
	// the confirmation response is handled in the rpc
	assert.Equal(t, int32(1), msgCount.Load())

	err = co1.ObserveProperty("", "")
	assert.NoError(t, err)
	err = co1.UnobserveProperty("", "")
	assert.NoError(t, err)
}
