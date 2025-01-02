package tests

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// test event messages between agent, server and client
// this uses the client and server helpers defined in connect_test.go

// Test subscribing and receiving all events by consumer
func TestSubscribeAll(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var rxVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var agentID = "agent1"
	var agentID2 = "agent2"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the servers
	srv, cancelFn, cm := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	// 2. connect as consumers
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cl1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	cl2 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cl2.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cl2.Disconnect()

	// ensure that agents can also subscribe (they cant use forms)
	ag2 := NewAgent(agentID2)
	_, err = ag2.ConnectWithPassword(agentID2)
	require.NoError(t, err)
	defer cl2.Disconnect()

	// FIXME: test subscription by agent

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	cl1.SetNotificationHandler(func(ev transports.NotificationMessage) {
		slog.Info("client 1 receives event")
		// receive event
		rxVal.Store(ev.Data)
		cancelFn()
	})
	cl2.SetNotificationHandler(func(ev transports.NotificationMessage) {
		slog.Info("client 2 receives event")
	})
	ag2.SetNotificationHandler(func(ev transports.NotificationMessage) {
		// receive event
		slog.Info("Agent receives event")
	})

	// Subscribe to events. Each binding implements this as per its spec
	err = cl1.Subscribe("", "")
	assert.NoError(t, err)
	err = cl2.Subscribe(thingID, eventKey)
	assert.NoError(t, err)
	err = ag2.Subscribe("", "")
	assert.NoError(t, err)

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	notif1 := transports.NewNotificationMessage(wot.HTOpEvent, thingID, eventKey, testMsg1)
	notif1.SenderID = agentID
	cm.PublishNotification(notif1)

	// 4. subscriber should have received them
	<-ctx.Done()
	assert.Equal(t, testMsg1, rxVal.Load())

	// Unsubscribe from events
	err = cl1.Unsubscribe("", "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // async take time

	err = cl2.Unsubscribe(thingID, eventKey)
	assert.NoError(t, err)
	err = ag2.Unsubscribe("", "")
	assert.NoError(t, err)

	// 5. Server sends another event to consumers
	notif2 := transports.NewNotificationMessage(wot.HTOpEvent, thingID, eventKey, testMsg2)
	notif2.SenderID = agentID
	cm.PublishNotification(notif2)
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
	var agentID = "agent1"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	// handler of event notification on the server
	notificationHandler := func(msg transports.NotificationMessage) {
		evVal.Store(msg.Data)
	}
	srv, cancelFn, _ := StartTransportServer(notificationHandler, nil, nil)
	_ = srv
	defer cancelFn()

	// 2. connect as an agent
	ag1 := NewAgent(testAgentID1)
	_, err := ag1.ConnectWithPassword(testAgentID1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent publishes an event
	notif := transports.NewNotificationMessage(wot.HTOpEvent, thingID, eventKey, testMsg)
	notif.SenderID = agentID
	err = ag1.SendNotification(notif)
	time.Sleep(time.Millisecond) // time to take effect
	require.NoError(t, err)

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}
