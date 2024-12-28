package tests

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

// test TD messages and forms
// this uses the client and server helpers defined in connect_test.go

const DeviceTypeSensor = "hiveot:sensor"

// Test subscribing a TD to the server by the agent
func TestPublishTDByAgent(t *testing.T) {
	t.Log("TestPublishTDByAgent")
	var evVal atomic.Value
	var thingID = "thing1"

	// notification handler of TDs on the server
	notificationHandler := func(msg transports.NotificationMessage) {
		evVal.Store(msg.Data)
	}

	// 1. start the transport
	_, cancelFn, _ := StartTransportServer(nil, nil, notificationHandler)
	defer cancelFn()

	// 2. connect as an agent
	ag1 := NewAgent(testAgentID1)
	_, err := ag1.ConnectWithPassword(testAgentID1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent creates TD
	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
	td1JSON, _ := jsoniter.Marshal(td1)
	// 4. agent publishes the TD
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, thingID, "", string(td1JSON))
	err = ag1.SendNotification(notif1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // time to take effect

	// TD received by server
	var rxMsg2Raw = evVal.Load()
	require.NotNil(t, rxMsg2Raw)
	rxMsg2 := rxMsg2Raw.(string)

	var td2 td.TD
	err = jsoniter.UnmarshalFromString(rxMsg2, &td2)
	require.NoError(t, err)
	assert.Equal(t, td1.ID, td2.ID)
	assert.Equal(t, td1.Title, td2.Title)
	assert.Equal(t, td1.AtType, td2.AtType)
}

// Test if forms are indeed added to a TD, describing the transport protocol binding operations
func TestAddForms(t *testing.T) {
	t.Log("TestPublishTDByAgent")
	var thingID = "thing1"

	// handler of TDs on the server
	// 1. start the transport
	_, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	// 2. Create a TD
	tdi := td.NewTD(thingID, "My gadget", DeviceTypeSensor)

	// 3. add forms
	err := transportServer.AddTDForms(tdi)
	require.NoError(t, err)

	// 4. Check that at least 1 form are present
	assert.GreaterOrEqual(t, len(tdi.Forms), 1)
}

func TestReadTD(t *testing.T) {
	t.Log("TestReadTD")
	var thingID = "thing1"

	// 2. Create a TD
	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)

	// handler of TDs on the server
	requestHandler := func(msg transports.RequestMessage, replyTo string) transports.ResponseMessage {
		resp := msg.CreateResponse(td1, nil)
		return resp
	}

	// 1. start the transport
	srv, cancelFn, _ := StartTransportServer(requestHandler, nil, nil)
	defer cancelFn()

	// 2. add forms
	err := transportServer.AddTDForms(td1)
	require.NoError(t, err)

	// 4. Check that at least 1 form are present
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cl1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)

	err = cl1.Subscribe(thingID, "")
	require.NoError(t, err)

	//td2, err := cl1.ReadTD(thingID)
	var td2 td.TD
	// FIXME: this requires a request handler
	err = cl1.Rpc(wot.HTOpReadTD, thingID, "", thingID, &td2)
	require.NoError(t, err)
	require.Equal(t, thingID, td2.ID)

	// cl1 should receive update to published TD
	var rxTD atomic.Bool
	cl1.SetNotificationHandler(func(msg transports.NotificationMessage) {
		rxTD.Store(true)
	})
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateTD, thingID, "", td2)
	srv.SendNotification(notif1)
	time.Sleep(time.Millisecond * 10)
	assert.True(t, rxTD.Load())
}
