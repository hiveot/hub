package tests

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

// test TD messages and forms
// this uses the client and server helpers defined in connect_test.go

// Test subscribing a TD to the server by the agent
func TestPublishTDByAgent(t *testing.T) {
	t.Log("TestPublishTDByAgent")
	var evVal atomic.Value
	var thingID = "thing1"

	// handler of TDs on the server
	handler1 := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) {
		// event handlers do not reply
		require.Nil(t, replyTo)
		evVal.Store(msg.Data)
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
	td1 := td.NewTD(thingID, "My gadget", vocab.ThingDevice)

	// 4. agent publishes the TD
	err = ag1.SendNotification(vocab.HTOpUpdateTD, thingID, "", td1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// TD received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)

	var td2 td.TD
	err = utils.Decode(rxMsg2, &td2)
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
	td := td.NewTD(thingID, "My gadget", vocab.ThingDevice)

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
	td1 := td.NewTD(thingID, "My gadget", vocab.ThingDevice)

	// handler of TDs on the server
	handler1 := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) {
		// event handlers do not reply
		output := td1
		replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
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

	//td2, err := cl1.ReadTD(thingID)
	var td2 td.TD
	err = cl1.SendRequest(wot.HTOpReadTD, thingID, "", thingID, &td2)
	require.NoError(t, err)
	require.Equal(t, thingID, td2.ID)

	// cl1 should receive update to published TD
	rxTD := false
	cl1.SetNotificationHandler(func(msg *transports.ThingMessage) {
		rxTD = true
	})
	srv.SendNotification(wot.HTOpUpdateTD, thingID, "", td2)
	time.Sleep(time.Millisecond * 10)
	assert.True(t, rxTD)
}
