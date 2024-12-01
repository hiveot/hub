package tests

import (
	"context"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

// test event messages between agent, server and client
// this uses the client and server helpers defined in connect_test.go

// Test subscribing and receiving all events by consumer
func TestSubscribeAllByConsumer(t *testing.T) {
	t.Log("TestSubscribeAllByConsumer")
	var rxVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the servers
	cancelFn, cm := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// 2. connect as a consumer
	cl1 := NewClient(testClientID1)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	cl1.SetMessageHandler(func(ev *transports.ThingMessage) {
		// receive event
		rxVal.Store(ev.Data)
		cancelFn()
	})

	// Subscribe to events
	form := NewForm(vocab.OpSubscribeAllEvents)
	_, err = cl1.SendOperation(form, "", "", nil, nil, "")
	require.NoError(t, err)
	// No result is expected

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	cm.PublishEvent(thingID, eventKey, testMsg1, "", testAgentID1)

	// 4. subscriber should have received them
	<-ctx.Done()
	assert.Equal(t, testMsg1, rxVal.Load())

	// Unsubscribe from events
	form = NewForm(vocab.OpUnsubscribeAllEvents)
	_, err = cl1.SendOperation(form, "", "", nil, nil, "")
	time.Sleep(time.Millisecond * 10) // async take time

	// 5. Server sends another event to consumers
	cm.PublishEvent(thingID, eventKey, testMsg2, "", testAgentID1)
	// update not received
	assert.Equal(t, testMsg1, rxVal.Load(), "Unsubscribe didnt work")

	//
}

// Agent sends events to server
// This is used if the Thing agent is connected as a client, and does not
// run a server itself.
func TestPublishEventsByAgent(t *testing.T) {
	t.Log("TestPublishEventsByAgent")
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// handler of events on the server
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

	// 3. agent publishes an event
	form := NewForm(vocab.HTOpPublishEvent)
	require.NotNil(t, form)
	status, err := ag1.SendOperation(form, thingID, eventKey, testMsg, nil, "")
	time.Sleep(time.Millisecond) // time to take effect
	require.NoError(t, err)

	// no reply is expected
	require.Equal(t, transports.RequestPending, status)

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}
