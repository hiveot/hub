// test property messages between the protocol client and server
package tests

import (
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
	srv, cancelFn, cm := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	// 2. connect with two consumers
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cl1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cl1.Disconnect()
	cl2 := NewConsumer(testClientID1, srv.GetForm)
	_, err = cl2.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	defer cl2.Disconnect()

	// set the handler for property updates and subscribe
	cl1.SetNotificationHandler(func(ev transports.NotificationMessage) {
		rxVal1.Store(ev.Data)
	})
	cl2.SetNotificationHandler(func(ev transports.NotificationMessage) {
		rxVal2.Store(ev.Data)
	})

	// Client1 subscribes to one, client 2 to all property updates
	err = cl1.ObserveProperty(thingID, propertyKey1)
	require.NoError(t, err)
	err = cl2.ObserveProperty("", "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// 3. Server sends a property update to consumers
	notif1 := transports.NewNotificationMessage(
		wot.HTOpUpdateProperty, thingID, propertyKey1, propValue1)
	notif1.SenderID = agentID
	cm.PublishNotification(notif1)

	// 4. both observers should have received it
	time.Sleep(time.Millisecond)
	assert.Equal(t, propValue1, rxVal1.Load())
	assert.Equal(t, propValue1, rxVal2.Load())

	// 5. client 1 unobserves
	err = cl1.UnobserveProperty(thingID, propertyKey1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // time to take effect

	// 6. Server sends a property update to consumers
	notif2 := transports.NewNotificationMessage(
		wot.HTOpUpdateProperty, thingID, propertyKey1, propValue2)
	notif2.SenderID = agentID
	cm.PublishNotification(notif2)
	notif3 := transports.NewNotificationMessage(
		wot.HTOpUpdateProperty, thingID, propertyKey2, propValue2)
	notif3.SenderID = agentID
	cm.PublishNotification(notif3)

	// 7. property should not have been received
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, propValue1, rxVal1.Load())
	assert.Equal(t, propValue2, rxVal2.Load())

	// 8. client 2 unobserves
	err = cl2.UnobserveProperty("", "")
	time.Sleep(time.Millisecond * 10)
	notif4 := transports.NewNotificationMessage(
		wot.HTOpUpdateProperty, thingID, propertyKey2, propValue1)
	notif4.SenderID = agentID
	cm.PublishNotification(notif4)
	// no change is expected
	assert.Equal(t, propValue2, rxVal2.Load())

}

// Agent sends property updates to server
// This is used if the Thing agent is connected as a client, and does not
// run a server itself.
func TestPublishPropertyByAgent(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var evVal atomic.Value
	var agentID = "agent1"
	var thingID = "thing1"
	var propKey1 = "property1"
	var propValue1 = "value1"

	// handler of property updates on the server
	notificationHandler := func(msg transports.NotificationMessage) {
		evVal.Store(msg.Data)
	}

	// 1. start the transport
	srv, cancelFn, _ := StartTransportServer(notificationHandler, nil, nil)
	_ = srv
	defer cancelFn()

	// 2. connect as an agent
	ag1 := NewAgent(testAgentID1)
	_, err := ag1.ConnectWithPassword(testAgentID1)
	require.NoError(t, err)
	defer ag1.Disconnect()

	// 3. agent publishes a property update
	notif1 := transports.NewNotificationMessage(wot.HTOpUpdateProperty, thingID, propKey1, propValue1)
	notif1.SenderID = agentID
	err = ag1.SendNotification(notif1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// property received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, propValue1, rxMsg2)
}
