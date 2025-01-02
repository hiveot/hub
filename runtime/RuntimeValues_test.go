package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryActions(t *testing.T) {
	t.Log("TestQueryActions")
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"
	const actionID = "action-0"
	var updateChan1 = make(chan bool)

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
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = td.MakeDigiTwinThingID(agentID, td1.ID)
	ag1.SetRequestHandler(func(msg transports.RequestMessage) transports.ResponseMessage {
		slog.Info("request: "+msg.Operation, "correlationID", msg.CorrelationID)
		return msg.CreateResponse(data, nil)
	})
	cl1.SetNotificationHandler(func(msg transports.NotificationMessage) {
		slog.Info("notification: " + msg.Operation)
		// signal notification received
		updateChan1 <- true
	})
	err := cl1.Subscribe("", "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	//err := ag1.PubTD(td1.ID, string(td1JSON))

	// step 2: consumer publish an action to the agent it should return as
	// a notification.
	notif := transports.NewNotificationMessage(wot.HTOpUpdateTD, td1.ID, "", string(td1JSON))
	err = ag1.SendNotification(notif)
	require.NoError(t, err)
	<-updateChan1

	// this action is recorded
	err = cl1.InvokeAction(dThing1ID, actionID, data, nil)
	require.NoError(t, err)

	// get the latest action values from the thing
	// use the API generated from the digitwin TD document using tdd2api
	//valueList, err := digitwin.ValuesQueryAllActions(cl1, dThing1ID)
	//require.NoError(t, err)
	//valueMap := api.ActionListToMap(valueList)

	// value must match that of the action in step 1 and match its correlationID
	//actVal := valueMap[actionID]
	//assert.Equal(t, data, actVal.Input)
}

// Get events from the outbox using the experimental http REST api
func TestReadEvents(t *testing.T) {
	t.Log("TestReadEvents")
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
	hc1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer hc1.Disconnect()

	// step 1: agent publishes a TD first: dtw:agent1:thing-1
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)
	var dThing1ID = td.MakeDigiTwinThingID(agentID, td1.ID)
	// is requested. hiveot uses it to determine if a response is required.
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, td1.ID, "", string(td1JSON))
	err := ag1.SendNotification(notif1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)

	notif2 := transports.NewNotificationMessage(wot.HTOpEvent, td1.ID, key1, data)
	err = ag1.SendNotification(notif2)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 1)

	dtwValues := make([]digitwin.ThingValue, 0)
	//stat := hc1.ReadAllEvents(dThing1ID, &dtwValues, "")
	// FIXME: the format of this operation is not defined by WoT or the
	// protocol binding spec. In this case the digitwin determines the output
	// format. (a list of ThingValue objects)
	err = hc1.Rpc(wot.HTOpReadAllEvents, dThing1ID, "", nil, &dtwValues)
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
	t.Log("TestHttpsGetProps")
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
	var dThingID = td.MakeDigiTwinThingID(agentID, td1.ID)
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, td1.ID, "", string(td1JSON))
	err := ag1.SendNotification(notif1)
	time.Sleep(time.Millisecond * 10)
	require.NoError(t, err)

	//err = ag1.PubProperty(td1.ID, key1, data1)
	//err = ag1.PubProperty(td1.ID, key2, data2)
	notif2 := transports.NewNotificationMessage(wot.HTOpUpdateProperty, td1.ID, key1, data1)
	err = ag1.SendNotification(notif2)
	notif3 := transports.NewNotificationMessage(wot.HTOpUpdateProperty, td1.ID, key2, data2)
	err = ag1.SendNotification(notif3)

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

	// 1. consumer subscribes to events/properties
	err := cl1.Subscribe("", "")
	require.NoError(t, err)
	cl1.SetNotificationHandler(func(msg transports.NotificationMessage) {
		msgCount.Add(1)
	})

	// 2: agent creates a TD first: dtw:agent1:thing-1 - this sends a notification
	td1 := ts.CreateTestTD(0)
	td1JSON, _ := json.Marshal(td1)

	// 3: agent publishes notification
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, td1.ID, "", string(td1JSON))
	err = ag1.SendNotification(notif1)

	time.Sleep(time.Millisecond * 100)

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2

	notif2 := transports.NewNotificationMessage(wot.HTOpEvent, td1.ID, key1, data1)
	err = ag1.SendNotification(notif2)
	notif3 := transports.NewNotificationMessage(wot.HTOpEvent, td1.ID, key2, data2)
	err = ag1.SendNotification(notif3)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)
	// one thing updated and two notification events
	assert.Equal(t, int32(3), msgCount.Load())
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
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, td1.ID, "", string(td1JSON))
	err := ag1.SendNotification(notif1)

	// agents listen for property write requests
	ag1.SetRequestHandler(func(msg transports.RequestMessage) transports.ResponseMessage {
		if msg.Operation == vocab.OpWriteProperty && msg.Name == key1 {
			msgCount.Add(1)
		}
		return msg.CreateResponse(nil, nil)
	})

	// consumer subscribes to events/properties changes
	err = cl1.ObserveProperty("", "")
	require.NoError(t, err)

	cl1.SetNotificationHandler(func(msg transports.NotificationMessage) {
		// expect an action status message that is the result of invokeaction
		if msg.Name == key1 {
			msgCount.Add(1)
		}
	})
	time.Sleep(time.Millisecond * 100)

	dThingID := td.MakeDigiTwinThingID(agentID, td1.ID)
	//stat2 := cl1.WriteProperty(dThingID, key1, data1)
	//require.Empty(t, stat2.Error)
	err = cl1.Rpc(wot.OpWriteProperty, dThingID, key1, data1, nil)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)

	// write property results in a request on the agent
	// the confirmation response is handled in the rpc
	assert.Equal(t, int32(1), msgCount.Load())

	err = cl1.ObserveProperty("", "")
	assert.NoError(t, err)
	err = cl1.UnobserveProperty("", "")
	assert.NoError(t, err)
}
