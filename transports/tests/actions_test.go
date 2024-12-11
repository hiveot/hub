package tests

import (
	"context"
	"errors"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
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

	// the server will receive the action request and return an immediate result
	serverHandler := func(msg *transports.ThingMessage, replyTo string) (
		handled bool, output any, err error) {
		if msg.Operation == wot.OpInvokeAction {
			inputVal.Store(msg.Data)
			return true, msg.Data, nil
		} else {
			assert.Fail(t, "Not expecting this")
		}
		return true, nil, errors.New("unexpected message")
	}
	// 1. start the servers
	srv, cancelFn, _ := StartTransportServer(serverHandler)
	defer cancelFn()

	// 2. connect a client
	cl1 := NewClient(testClientID1, srv.GetForm)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	defer cl1.Disconnect()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	ctx1, release1 := context.WithTimeout(context.Background(), time.Minute)
	defer release1()
	// expect the client to receive an action/request status message as a notification
	cl1.SetNotificationHandler(func(ev *transports.ThingMessage) {
		// this depends on the transport protocol
		slog.Info("testOutput was updated asynchronously via the message handler")
		err2 := tputils.Decode(ev.Data, &testOutput)
		assert.NoError(t, err2)
		release1()
	})
	// 3. invoke the action as a notification, (not a rpc request)
	// testOutput can be updated as an immediate result or via the callback message handler
	err = cl1.SendNotification(wot.OpInvokeAction, thingID, actionName, testMsg1)
	assert.NoError(t, err)
	<-ctx1.Done()

	//time.Sleep(time.Millisecond * 10)
	// whether receiving completed or delivered depends on the binding
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
	var reqVal atomic.Value
	var replyVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var corrID = "correlation-1"

	// 1. start the server. register a message handler for receiving an action status
	// async reply from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.

	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn1()
	serverHandler := func(msg *transports.ThingMessage, replyTo string) (
		handled bool, output any, err error) {

		// The server receives an action status reply message from the agent
		// (which normally is forwarded to the remote consumer; but not in this test)
		assert.Empty(t, replyTo)
		assert.Equal(t, testAgentID1, msg.SenderID)
		require.Equal(t, wot.HTOpUpdateActionStatus, msg.Operation)

		slog.Info("serverHandler: Received reply message from agent",
			"op", msg.Operation,
			"data", msg.DataAsText(),
			"senderID", msg.SenderID)
		replyVal.Store(msg.Data)
		cancelFn1()
		// there is no result for this reply
		return true, nil, nil
	}
	srv, cancelFn, cm := StartTransportServer(serverHandler)
	defer cancelFn()

	// 2a. connect as an agent
	ag1client := NewClient(testAgentID1, srv.GetForm)
	token, err := ag1client.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	defer ag1client.Disconnect()

	// an agent receives requests from the server
	ag1client.SetRequestHandler(func(msg *transports.ThingMessage) (output any, err error) {
		// agent receives action request and returns a result
		slog.Info("Agent receives message", "op", msg.Operation)
		assert.Equal(t, testClientID1, msg.SenderID)
		reqVal.Store(msg.Data)
		return testMsg2, nil
	})

	// send the action from the server to the agent (the agent is connected as a client)
	// and expect result using the request status message sent by the agent.
	time.Sleep(time.Millisecond)
	ag1Server := cm.GetConnectionByClientID(testAgentID1)
	require.NotNil(t, ag1Server)
	msg := transports.NewThingMessage(wot.OpInvokeAction, thingID, actionKey, testMsg1, corrID)
	msg.SenderID = testClientID1
	ag1Server.SendRequest(*msg)

	// wait until the agent has sent a reply
	<-ctx1.Done()

	// if all went well the agent received the request and the server its response
	assert.Equal(t, testMsg1, reqVal.Load())
	assert.Equal(t, testMsg2, replyVal.Load())
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
	serverHandler := func(msg *transports.ThingMessage, replyTo string) (
		handled bool, output any, err error) {

		require.NotNil(t, replyTo)
		// FIXME: what is the format of a action query result
		if msg.Operation == wot.OpQueryAction {
			// the reply is an action status instance
			output := transports.RequestStatus{
				ThingID:       msg.ThingID,
				Name:          msg.Name,
				RequestID:     msg.RequestID,
				Status:        transports.StatusCompleted,
				Output:        testMsg1,
				TimeRequested: msg.Timestamp,
				TimeEnded:     time.Now().Format(wot.RFC3339Milli),
			}
			return true, output, nil
			//replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
		} else if msg.Operation == wot.OpQueryAllActions {
			actStat := make([]transports.RequestStatus, 2)
			actStat[0].ThingID = thingID
			actStat[0].Name = actionKey
			actStat[0].Output = testMsg1
			actStat[1].ThingID = thingID
			actStat[1].Name = actionKey
			actStat[1].Output = testMsg1
			output = actStat
			return true, output, nil
			//replyTo.SendResponse(msg.ThingID, msg.Name, actStat, msg.RequestID)
		}
		return true, nil, errors.New("unexpected request " + msg.Operation)
	}

	// 1. start the servers
	srv, cancelFn, _ := StartTransportServer(serverHandler)
	defer cancelFn()

	// 2. connect as a consumer
	cl1 := NewClient(testClientID1, srv.GetForm)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// 3. Query action status
	var output transports.RequestStatus
	err = cl1.SendRequest(wot.OpQueryAction, thingID, actionKey, nil, &output)
	require.NoError(t, err)
	require.Equal(t, thingID, output.ThingID)
	require.Equal(t, actionKey, output.Name)

	// 4. Query all actions
	var output2 []transports.RequestStatus
	err = cl1.SendRequest(wot.OpQueryAllActions, thingID, actionKey, nil, &output2)
	require.NoError(t, err)
	require.Equal(t, 2, len(output2))
}
