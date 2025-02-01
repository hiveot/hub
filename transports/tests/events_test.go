package tests

import (
	"context"
	"errors"
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
	var thingID = "thing1"
	var eventKey = "event11"
	var agentRxEvent atomic.Bool

	// 1. start the servers
	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()

	// 2. connect as consumers
	cconn1, cons1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cconn1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cconn1.Disconnect()

	cconn2, cons2 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cconn2.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cconn2.Disconnect()

	// ensure that agents can also subscribe (they cant use forms)
	agConn1, agent1 := NewAgent(agentID)
	_, err = agConn1.ConnectWithPassword(agentID)
	require.NoError(t, err)
	defer agConn1.Disconnect()

	// FIXME: test subscription by agent

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	cons1.SetResponseHandler(func(ev *transports.ResponseMessage) error {
		slog.Info("client 1 receives event")
		// receive event
		rxVal.Store(ev.Output)
		cancelFn()
		return nil
	})
	cons2.SetResponseHandler(func(ev *transports.ResponseMessage) error {
		slog.Info("client 2 receives event")
		return nil
	})
	agent1.SetResponseHandler(func(ev *transports.ResponseMessage) error {
		// receive event
		slog.Info("Agent receives event")
		agentRxEvent.Store(true)
		return nil
	})

	// Subscribe to events. Each binding implements this as per its spec
	err = cons1.Subscribe("", "")
	assert.NoError(t, err)
	err = cons2.Subscribe(thingID, eventKey)
	assert.NoError(t, err)
	err = agent1.Subscribe("", "")
	assert.NoError(t, err)

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	notif1 := transports.NewNotificationResponse(wot.OpSubscribeEvent, thingID, eventKey, testMsg1, nil)
	notif1.SenderID = agentID
	srv.SendNotification(notif1)

	// 4. subscriber should have received them
	<-ctx.Done()
	assert.Equal(t, testMsg1, rxVal.Load())
	time.Sleep(time.Millisecond)
	assert.True(t, agentRxEvent.Load())

	// Unsubscribe from events
	err = cons1.Unsubscribe("", "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // async take time
	err = cons2.Unsubscribe(thingID, eventKey)
	assert.NoError(t, err)
	err = agent1.Unsubscribe("", "")
	assert.NoError(t, err)
	agentRxEvent.Store(false)

	// 5. Server sends another event to consumers
	notif2 := transports.NewNotificationResponse(wot.OpSubscribeEvent, thingID, eventKey, testMsg2, nil)
	notif2.SenderID = agentID
	srv.SendNotification(notif2)
	time.Sleep(time.Millisecond)
	// update not received
	assert.Equal(t, testMsg1, rxVal.Load(), "Unsubscribe didnt work")
	assert.False(t, agentRxEvent.Load())

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

	// 1. start the transport
	// handler of event notification on the server
	notificationHandler := func(msg *transports.ResponseMessage) error {
		evVal.Store(msg.Output)
		return nil
	}
	srv, cancelFn := StartTransportServer(nil, notificationHandler)
	_ = srv
	defer cancelFn()

	// 2. connect as an agent
	agConn1, agent1 := NewAgent(testAgentID1)
	_, err := agConn1.ConnectWithPassword(testAgentID1)
	require.NoError(t, err)
	defer agConn1.Disconnect()

	// 3. agent publishes an event
	err = agent1.PubEvent(thingID, eventKey, testMsg)
	time.Sleep(time.Millisecond) // time to take effect
	require.NoError(t, err)

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}

// Consumer reads events from agent
func TestReadEvent(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var thingID = "thing1"
	var eventKey = "event11"
	var eventValue = "value11"
	var timestamp = "eventtime"

	// 1. start the agent transport with the request handler
	// in this case the consumer connects to the agent (unlike when using a hub)
	agentReqHandler := func(req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
		if req.Operation == wot.HTOpReadEvent && req.ThingID == thingID && req.Name == eventKey {
			evVal := transports.ThingValue{
				ID:      "ud1",
				Name:    req.Name,
				Output:  eventValue,
				ThingID: thingID,
				Updated: timestamp,
			}
			resp := req.CreateResponse(evVal, nil)
			resp.Updated = timestamp
			return resp
		}
		return req.CreateResponse(nil, errors.New("unexpected request"))
	}
	srv, cancelFn := StartTransportServer(agentReqHandler, nil)
	_ = srv
	defer cancelFn()

	// 2. connect as a consumer
	cc1, consumer1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cc1.Disconnect()

	rxVal, err := consumer1.ReadEvent(thingID, eventKey)
	require.NoError(t, err)
	assert.Equal(t, eventValue, rxVal.Output)
	assert.Equal(t, timestamp, rxVal.Updated)
}

// Consumer reads events from agent
func TestReadAllEvents(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var thingID = "thing1"
	var event1Name = "event1"
	var event2Name = "event2"
	var event1Value = "value1"
	var event2Value = "value2"

	// 1. start the agent transport with the request handler
	// in this case the consumer connects to the agent (unlike when using a hub)
	agentReqHandler := func(req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
		if req.Operation == wot.HTOpReadAllEvents {
			output := make(map[string]*transports.ResponseMessage)
			output[event1Name] = transports.NewResponseMessage(wot.OpSubscribeEvent, thingID, event1Name, event1Value, nil, "")
			output[event2Name] = transports.NewResponseMessage(wot.OpSubscribeEvent, thingID, event2Name, event2Value, nil, "")
			resp := req.CreateResponse(output, nil)
			return resp
		}
		return req.CreateResponse(nil, errors.New("unexpected request"))
	}
	srv, cancelFn := StartTransportServer(agentReqHandler, nil)
	_ = srv
	defer cancelFn()

	// 2. connect as a consumer
	cc1, consumer1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cc1.Disconnect()

	evMap, err := consumer1.ReadAllEvents(thingID)
	require.NoError(t, err)
	require.Equal(t, 2, len(evMap))
	require.Equal(t, event1Value, evMap[event1Name].Output)
	require.Equal(t, event2Value, evMap[event2Name].Output)
}
