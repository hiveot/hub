package tests

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
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
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var rxVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the servers
	srv, cancelFn, cm := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// 2. connect as consumers
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()
	cl2 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cl2.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl2.Disconnect()

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	cl1.SetNotificationHandler(func(ev *transports.ThingMessage) {
		// receive event
		rxVal.Store(ev.Data)
		cancelFn()
	})

	// Subscribe to events. Each binding implements this as per its spec
	err = cl1.Subscribe("", "")
	assert.NoError(t, err)
	err = cl2.Subscribe(thingID, eventKey)
	assert.NoError(t, err)

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	cm.PublishEvent(thingID, eventKey, testMsg1, "", testAgentID1)

	// 4. subscriber should have received them
	<-ctx.Done()
	assert.Equal(t, testMsg1, rxVal.Load())

	// Unsubscribe from events
	err = cl1.Unsubscribe("", "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // async take time

	err = cl2.Unsubscribe(thingID, eventKey)

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
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// handler of events on the server
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
	ag1 := NewAgent(testAgentID1, srv.GetForm)
	_, err := ag1.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent publishes an event
	err = ag1.SendNotification(wot.HTOpPublishEvent, thingID, eventKey, testMsg)
	time.Sleep(time.Millisecond) // time to take effect
	require.NoError(t, err)

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}
