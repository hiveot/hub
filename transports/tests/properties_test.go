// test property messages between the protocol client and server
package tests

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

// test property messages between agent, server and client
// this uses the client and server helpers defined in connect_test.go

// Test observing and receiving all properties by consumer
func TestObservePropertyByConsumer(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var rxVal1 atomic.Value
	var rxVal2 atomic.Value
	var agentID = "agent1"
	var thingID = "thing1"
	var propertyKey1 = "property1"
	var propertyKey2 = "property2"
	var propValue1 = "value1"
	var propValue2 = "value2"

	// 1. start the server
	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()

	// 2. connect with two consumers
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cc1.Disconnect()
	cc2, cl2 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cc2.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cc2.Disconnect()

	// set the handler for property updates and subscribe
	cl1.SetResponseHandler(func(ev *transports.ResponseMessage) error {
		rxVal1.Store(ev.Output)
		return nil
	})
	cl2.SetResponseHandler(func(ev *transports.ResponseMessage) error {
		rxVal2.Store(ev.Output)
		return nil
	})

	// Client1 subscribes to one, client 2 to all property updates
	err = cl1.ObserveProperty(thingID, propertyKey1)
	require.NoError(t, err)
	err = cl2.ObserveProperty("", "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// 3. Server sends a property update to consumers
	notif1 := transports.NewNotificationResponse(wot.OpObserveProperty, thingID, propertyKey1, propValue1, nil)
	notif1.SenderID = agentID
	srv.SendNotification(notif1)

	// 4. both observers should have received it
	time.Sleep(time.Millisecond)
	assert.Equal(t, propValue1, rxVal1.Load())
	assert.Equal(t, propValue1, rxVal2.Load())

	// 5. client 1 unobserves
	err = cl1.UnobserveProperty(thingID, propertyKey1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // time to take effect

	// 6. Server sends a property update to consumers
	notif2 := transports.NewNotificationResponse(wot.OpObserveProperty, thingID, propertyKey1, propValue2, nil)
	notif2.SenderID = agentID
	srv.SendNotification(notif2)
	notif3 := transports.NewNotificationResponse(wot.OpObserveProperty, thingID, propertyKey2, propValue2, nil)
	notif3.SenderID = agentID
	srv.SendNotification(notif3)

	// 7. property should not have been received
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, propValue1, rxVal1.Load())
	assert.Equal(t, propValue2, rxVal2.Load())

	// 8. client 2 unobserves
	err = cl2.UnobserveProperty("", "")
	time.Sleep(time.Millisecond * 10)
	notif4 := transports.NewNotificationResponse(wot.OpObserveProperty, thingID, propertyKey2, propValue1, nil)
	notif4.SenderID = agentID
	srv.SendNotification(notif4)
	// no change is expected
	assert.Equal(t, propValue2, rxVal2.Load())

}

// Agent sends property updates to server
// This is used if the Thing agent is connected as a client, and does not
// run a server itself.
func TestPublishPropertyByAgent(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var evVal atomic.Value
	var thingID = "thing1"
	var propKey1 = "property1"
	var propValue1 = "value1"

	// handler of property updates on the server
	notificationHandler := func(msg *transports.ResponseMessage) error {
		evVal.Store(msg.Output)
		return nil
	}

	// 1. start the transport
	srv, cancelFn := StartTransportServer(nil, notificationHandler)
	_ = srv
	defer cancelFn()

	// 2. connect as an agent
	agConn1, ag1 := NewAgent(testAgentID1)
	_, err := agConn1.ConnectWithPassword(testAgentID1)
	require.NoError(t, err)
	defer agConn1.Disconnect()

	// 3. agent publishes a property update
	err = ag1.PubProperty(thingID, propKey1, propValue1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// property received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, propValue1, rxMsg2)
}

// Consumer reads property from agent
func TestReadProperty(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var thingID = "thing1"
	var propKey = "propKey1"
	var propValue = "value11"

	// 1. start the agent transport with the request handler
	// in this case the consumer connects to the agent (unlike when using a hub)
	agentReqHandler := func(req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
		if req.Operation == wot.OpReadProperty && req.ThingID == thingID && req.Name == propKey {
			return req.CreateResponse(propValue, nil)
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

	rxVal, err := consumer1.ReadProperty(thingID, propKey)
	require.NoError(t, err)
	assert.Equal(t, propValue, rxVal)
}
