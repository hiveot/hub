package tests

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
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
	handler1 := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {
		// event handlers do not reply
		require.Nil(t, replyTo)
		evVal.Store(msg.Data)
		return stat
	}

	// 1. start the transport
	cancelFn, _ := StartTransportServer(handler1)
	defer cancelFn()

	// 2. connect as an agent
	ag1 := NewClient(testAgentID1)
	_, err := ag1.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent creates TD
	td := tdd.NewTD(thingID, "My gadget", vocab.ThingDevice)

	// 4. agent publishes the TD
	form := NewForm(vocab.HTOpUpdateTD)
	require.NotNil(t, form)
	status, err := ag1.SendOperation(form, thingID, "", td, nil, "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// no reply is expected
	require.Equal(t, transports.RequestPending, status)

	// TD received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)

	var td2 tdd.TD
	err = utils.Decode(rxMsg2, &td2)
	assert.Equal(t, td.ID, td2.ID)
	assert.Equal(t, td.Title, td2.Title)
	assert.Equal(t, td.AtType, td2.AtType)
}

// Test if forms are indeed added to a TD, describing the transport protocol binding operations
func TestAddForms(t *testing.T) {
	t.Log("TestPublishTDByAgent")
	var thingID = "thing1"

	// handler of TDs on the server
	// 1. start the transport
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// 2. Create a TD
	td := tdd.NewTD(thingID, "My gadget", vocab.ThingDevice)

	// 3. add forms
	err := transportServer.AddTDForms(td)
	require.NoError(t, err)

	// 4. Check that at least 1 form are present
	assert.GreaterOrEqual(t, len(td.Forms), 1)
}
