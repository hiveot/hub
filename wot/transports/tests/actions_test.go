package tests

import (
	"context"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/transports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// TestInvokeActionFromConsumerToServer: classic 'consumer talks to the server'
// as if it is a Thing. In this test the server replies.
// (routing is not part of this package)
func TestInvokeActionFromConsumerToServer(t *testing.T) {
	t.Log("TestInvokeActionFromConsumer")
	//var outputVal atomic.Value
	var testOutput string

	var inputVal atomic.Value
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionName = "action1"

	// the server will receive the action request and return with completed
	serverHandler := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {
		if msg.Operation == vocab.OpInvokeAction {
			require.NotNil(t, replyTo)
			inputVal.Store(msg.Data)
			// return the input as output
			stat.Completed(msg, msg.Data, nil)
		} else {
			assert.Fail(t, "Not expecting this")
		}
		return stat
	}
	// 1. start the servers
	cancelFn, _ := StartTransportServer(serverHandler)
	defer cancelFn()

	// 2. connect a client
	cl1 := NewConsumerClient(testClientID1)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	defer cl1.Disconnect()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	// expect the client to receive an action/request status message
	cl1.SetMessageHandler(func(ev *transports.ThingMessage) {
		// this depends on the transport protocol
		slog.Info("testOutput was updated asynchronously via the message handler")
		err = utils.Decode(ev.Data, &testOutput)
		assert.NoError(t, err)
	})
	// 3. invoke the action
	form := NewForm(vocab.OpInvokeAction)
	require.NotNil(t, form)

	// testOutput can be updated as an immediate result or via the callback message handler
	status, err := cl1.SendOperation(form, thingID, actionName, testMsg1, &testOutput, "")
	require.NoError(t, err)
	require.Equal(t, transports.RequestCompleted, status)
	require.Equal(t, testMsg1, testOutput)

	// 4. verify that the server received it and send a reply
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, inputVal.Load())
	assert.Equal(t, testMsg1, testOutput)
}

// Warning: this is a bit of a mind bender if you're used to classic consumer->thing interaction.
// This test uses a Thing agent as a client and have it reply to a request from the server.
// The server in this case passes on a message received from a consumer, which is also a client.
func TestInvokeActionFromServerToAgent(t *testing.T) {
	t.Log("TestPostAction")
	var rxVal atomic.Value
	var replyVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var corrID = "correlation-1"

	// 1. start the server. register a message handler for receiving an action status
	// reply from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.

	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn1()
	replyHandler := func(
		msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {

		// The server receives an action status reply message from the agent
		// (which normally is forwarded to the remote consumer; but not in this test)
		assert.Nil(t, replyTo)
		assert.Equal(t, testAgentID1, msg.SenderID)
		require.Equal(t, vocab.HTOpUpdateActionStatus, msg.Operation)

		// the reply is a RequestStatus instance, sent by the agent
		var rxStat transports.RequestStatus
		err := utils.Decode(msg.Data, &rxStat)
		require.NoError(t, err)

		slog.Info("replyHandler: Received reply message from agent",
			"op", msg.Operation,
			"data", msg.DataAsText(),
			"senderID", msg.SenderID)
		replyVal.Store(rxStat.Output)
		cancelFn1()
		return stat
	}
	cancelFn, cm := StartTransportServer(replyHandler)
	defer cancelFn()

	// 2a. connect as an agent
	ag1client := NewAgentClient(testAgentID1)
	token, err := ag1client.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	ag1client.SetRequestHandler(func(msg *transports.ThingMessage) (stat transports.RequestStatus) {
		// agent receives action request
		slog.Info("Agent receives message", "op", msg.Operation)
		rxVal.Store(msg.Data)
		// then returns its own output
		stat.Completed(msg, testMsg2, nil)
		return stat
	})

	// send the action from the server to the agent (the agent is connected as a client)
	// and expect result using the request status message sent by the agent.
	ag1Server := cm.GetConnectionByClientID(testAgentID1)
	require.NotNil(t, ag1Server)
	status, output, err := ag1Server.InvokeAction(thingID, actionKey, testMsg1, corrID, testClientID1)
	require.NoError(t, err)

	// wait until a reply is received
	<-ctx1.Done()

	assert.Equal(t, testMsg1, rxVal.Load())
	assert.Equal(t, testMsg2, replyVal.Load())

	// In protocol bindings the return channel is asynchronous from the request, so
	// the server receives the reply from the agent as a separate message.
	// This means that status is delivered and output is empty.
	// If however, there is a synchronous binding then this test will have to be updated.

	//assert.Equal(t, testMsg2, output)
	//assert.Equal(t, transports.RequestCompleted, status)
	assert.Equal(t, nil, output)
	assert.Equal(t, transports.RequestDelivered, status)
	ag1client.Disconnect()
}

// TestQueryActions consumer queries the server for actions
func TestQueryActions(t *testing.T) {
	t.Log("TestPostAction")
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionKey = "action1"

	// 1. start the server. register a message handler for receiving an action status
	// reply from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.
	queryHandler := func(
		msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {

		require.NotNil(t, replyTo)
		if msg.Operation == vocab.OpQueryAction {
			// the reply is an action status instance
			var actStat transports.RequestStatus
			actStat.ThingID = thingID
			actStat.Name = actionKey
			actStat.Output = testMsg1
			stat.Completed(msg, actStat, nil)
		} else if msg.Operation == vocab.OpQueryAllActions {
			var actStat = make([]transports.RequestStatus, 2)
			actStat[0].ThingID = thingID
			actStat[0].Name = actionKey
			actStat[0].Output = testMsg1
			actStat[1].ThingID = thingID
			actStat[1].Name = actionKey
			actStat[1].Output = testMsg1
			stat.Completed(msg, actStat, nil)
		}
		return stat
	}

	// 1. start the servers
	cancelFn, _ := StartTransportServer(queryHandler)
	defer cancelFn()

	// 2. connect as a consumer
	cl1 := NewConsumerClient(testClientID1)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// 3. Query action status
	form := NewForm(vocab.OpQueryAction)
	var output transports.RequestStatus
	status, err := cl1.SendOperation(form, thingID, actionKey, nil, &output, "")
	require.NoError(t, err)
	require.Equal(t, transports.RequestCompleted, status)
	require.Equal(t, thingID, output.ThingID)
	require.Equal(t, actionKey, output.Name)

	// 4. Query all actions
	form = NewForm(vocab.OpQueryAllActions)
	var output2 []transports.RequestStatus
	status, err = cl1.SendOperation(form, thingID, actionKey, nil, &output2, "")
	require.NoError(t, err)
	require.Equal(t, transports.RequestCompleted, status)
	require.Equal(t, 2, len(output2))
}
