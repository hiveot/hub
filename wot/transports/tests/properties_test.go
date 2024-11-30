// test property messages between the protocol client and server
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

// test property messages between agent, server and client
// this uses the client and server helpers defined in connect_test.go

// Test observing and receiving all properties by consumer
func TestObserveAllByConsumer(t *testing.T) {
	t.Log("TestObserveAllByConsumer")
	var rxVal atomic.Value
	var thingID = "thing1"
	var propertyKey1 = "property1"
	var propValue1 = "value1"
	var propValue2 = "value2"

	// 1. start the servers
	cancelFn, cm := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// 2. connect as a consumer
	cl1 := NewClient(testClientID1)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// set the handler for properties and subscribe
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn1()

	cl1.SetMessageHandler(func(ev *transports.ThingMessage) {
		t.Log("Received event: " + ev.Operation)
		// receive property update
		rxVal.Store(ev.Data)
		cancelFn1()
	})

	// Subscribe to property updates
	form := NewForm(vocab.OpObserveAllProperties)
	_, err = cl1.SendOperation(form, "", "", nil, nil, "")
	// No result is expected
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// 3. Server sends a property update to consumers
	cm.PublishProperty(thingID, propertyKey1, propValue1, "", testAgentID1)

	// 4. observer should have received them
	<-ctx1.Done()
	assert.Equal(t, propValue1, rxVal.Load())

	// 5. unobserve properties
	form = NewForm(vocab.OpUnobserveAllProperties)
	_, err = cl1.SendOperation(form, "", "", nil, nil, "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// 6. Server sends a property update to consumers
	cm.PublishProperty(thingID, propertyKey1, propValue2, "", testAgentID1)

	// 7. property should not have been received
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, propValue1, rxVal.Load())

}

// Agent sends property updates to server
// This is used if the Thing agent is connected as a client, and does not
// run a server itself.
func TestPublishPropertyByAgent(t *testing.T) {
	t.Log("TestPublishPropertyByAgent")
	var evVal atomic.Value
	var thingID = "thing1"
	var propKey1 = "property1"
	var propValue1 = "value1"

	// handler of property updates on the server
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

	// 3. agent publishes a property update
	form := NewForm(vocab.HTOpUpdateProperty)
	require.NotNil(t, form)
	status, err := ag1.SendOperation(form, thingID, propKey1, propValue1, nil, "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond) // time to take effect

	// no reply is expected
	require.Equal(t, transports.RequestPending, status)

	// property received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, propValue1, rxMsg2)
}
