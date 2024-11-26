package tests

import (
	"github.com/hiveot/hub/api/go/vocab"
	transports2 "github.com/hiveot/hub/wot/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// test event messages between protocol client and server

// Test posting an event and action
func TestPostEventAction(t *testing.T) {
	t.Log("TestPostEventAction")
	var rxVal atomic.Value
	var evVal atomic.Value
	var actVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	handleMessage := func(msg *transports2.ThingMessage, replyTo transports2.IServerConnection) (
		stat transports2.RequestStatus) {
		slog.Info("Received message", "op", msg.Operation)

		if msg.Operation == vocab.OpInvokeAction {
			require.NotNil(t, replyTo)
			actVal.Store(msg.Data)
			stat.Completed(msg, msg.Data, nil)

		} else {
			evVal.Store(msg.Data)
			stat.Delivered(msg)
		}
		return stat
	}
	// 1. start the servers
	cancelFn, _ := StartTransportServer(handleMessage)
	defer cancelFn()

	// 2a. connect as an agent and client
	ag1 := NewAgentClient(testAgentID1)
	token, err := ag1.ConnectWithPassword(testAgentPassword1)
	cl1 := NewConsumerClient(testClientID1)
	_, err = cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	cl1.SetMessageHandler(func(ev *transports2.ThingMessage) {
		// receive result from action
		rxVal.Store(ev.Data)
	})
	// generate the form for the action operation
	form := NewForm(vocab.OpSubscribeAllEvents)
	stat := cl1.SendOperation(form, "", "", nil, nil, "")
	require.Empty(t, stat.Error)
	//cl1.Subscribe("", "")

	// 3. publish two events
	//err = ag1.PubEvent(thingID, eventKey, testMsg1, "")
	//require.NoError(t, err)
	form = NewForm(vocab.HTOpPublishEvent)
	stat = ag1.SendOperation(form, thingID, eventKey, testMsg1, nil, "")
	require.Empty(t, stat.Error)
	stat = ag1.SendOperation(form, thingID, eventKey, testMsg1, nil, "")
	require.Empty(t, stat.Error)

	// 4. verify that the server received it
	time.Sleep(time.Millisecond * 100)
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, evVal.Load())

	// 5. publish an action
	//stat := cl1.InvokeAction(thingID, actionKey, testMsg2, nil, "")
	var reply any
	form = NewForm(vocab.OpInvokeAction)
	err = cl1.Rpc(form, thingID, actionKey, testMsg2, &reply)
	require.NoError(t, err)
	// response must be seen by router and received
	assert.Equal(t, testMsg2, actVal.Load())
	assert.Equal(t, testMsg2, reply)
	ag1.Disconnect()
}

// Test publish subscribe
func TestPubSub(t *testing.T) {
	t.Log("TestPubSub")
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// handler of events
	handler1 := func(msg *transports2.ThingMessage, replyTo transports2.IServerConnection) (
		stat transports2.RequestStatus) {
		evVal.Store(msg.Data)
		return stat
	}

	// 1. start the transport
	cancelFn, _ := StartTransportServer(handler1)
	defer cancelFn()

	// 2. connect with a client and agent
	cl1 := NewConsumerClient(testClientID1)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	ag1 := NewAgentClient(testAgentID1)
	_, err = ag1.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// ... and subscribe to events
	form := NewForm(vocab.OpSubscribeAllEvents)
	stat := cl1.SendOperation(form, thingID, "", nil, nil, "")
	require.Empty(t, stat.Error)

	// 4. publish an event using the hub client, the server will invoke the message dtwRouter
	// which in turn will publish this to the listeners, including this client.
	form = NewForm(vocab.HTOpPublishEvent)
	stat = cl1.SendOperation(form, thingID, eventKey, testMsg, nil, "")
	assert.Empty(t, stat.Error)
	time.Sleep(time.Millisecond * 10)
	//
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}
