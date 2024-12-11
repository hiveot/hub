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

	// handler of TDs on the server
	handler1 := func(msg *transports.ThingMessage, replyTo string) (
		handled bool, output any, err error) {
		// event handlers do not reply
		require.Empty(t, replyTo)
		evVal.Store(msg.Data)
		return true, nil, nil
	}

	// 1. start the transport
	srv, cancelFn, _ := StartTransportServer(handler1)
	defer cancelFn()

	// 2. connect as an agent
	ag1 := NewClient(testAgentID1, srv.GetForm)
	_, err := ag1.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent creates TD
	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
	td1JSON, _ := jsoniter.Marshal(td1)
	// 4. agent publishes the TD
	err = ag1.SendNotification(wot.HTOpUpdateTD, thingID, "", string(td1JSON))
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
	_, cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// 2. Create a TD
	td := td.NewTD(thingID, "My gadget", DeviceTypeSensor)

	// 3. add forms
	err := transportServer.AddTDForms(td)
	require.NoError(t, err)

	// 4. Check that at least 1 form are present
	assert.GreaterOrEqual(t, len(td.Forms), 1)
}

func TestReadTD(t *testing.T) {
	t.Log("TestReadTD")
	var thingID = "thing1"

	// 2. Create a TD
	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)

	// handler of TDs on the server
	handler1 := func(msg *transports.ThingMessage, replyTo string) (
		handled bool, output any, err error) {
		output = td1
		//replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
		return true, output, nil
	}

	// 1. start the transport
	srv, cancelFn, _ := StartTransportServer(handler1)
	defer cancelFn()

	// 3. add forms
	err := transportServer.AddTDForms(td1)
	require.NoError(t, err)

	// 4. Check that at least 1 form are present
	cl1 := NewClient(testClientID1, srv.GetForm)
	_, err = cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)

	cl1.Subscribe(thingID, "")

	//td2, err := cl1.ReadTD(thingID)
	var td2 td.TD
	err = cl1.SendRequest(wot.HTOpReadTD, thingID, "", thingID, &td2)
	require.NoError(t, err)
	require.Equal(t, thingID, td2.ID)

	// cl1 should receive update to published TD
	var rxTD atomic.Bool
	cl1.SetNotificationHandler(func(msg *transports.ThingMessage) {
		rxTD.Store(true)
	})
	srv.SendNotification(wot.HTOpUpdateTD, thingID, "", td2)
	time.Sleep(time.Millisecond * 10)
	assert.True(t, rxTD.Load())
}
