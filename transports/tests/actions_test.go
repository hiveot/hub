package tests

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// TestInvokeActionFromConsumerToServer: classic 'consumer talks to the server'
// as if it is a Thing. In this test the server replies.
// (routing is not part of this package)
func TestInvokeActionFromConsumerToServer(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	//var outputVal atomic.Value
	var testOutput string

	var inputVal atomic.Value
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionName = "action1"

	// the server will receive the action request and return an immediate result
	requestHandler := func(req transports.RequestMessage, replyTo string) transports.ResponseMessage {
		if req.Operation == wot.OpInvokeAction {
			inputVal.Store(req.Input)
			// Hmm, should this be pending with a separate async completed result?
			return req.CreateResponse(req.Input, nil)
		}
		assert.Fail(t, "Not expecting this")
		return req.CreateResponse(nil, errors.New("unexpected request"))
	}
	// 1. start the servers
	srv, cancelFn, _ := StartTransportServer(requestHandler, nil, nil)
	defer cancelFn()

	// 2. connect a client
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	defer cl1.Disconnect()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	ctx1, release1 := context.WithTimeout(context.Background(), time.Minute)
	defer release1()

	// since there is no waiting for a response when sending the request, the
	// client should receive an action/request response via the response callback
	cl1.SetResponseHandler(func(resp transports.ResponseMessage) {
		slog.Info("testOutput was updated asynchronously via the message handler")
		err2 := tputils.Decode(resp.Output, &testOutput)
		assert.NoError(t, err2)
		release1()
	})

	// 3. invoke the action without waiting for a result
	// the response handler above will receive the result
	// testOutput can be updated as an immediate result or via the callback message handler
	req := transports.NewRequestMessage(wot.OpInvokeAction, thingID, actionName, testMsg1, shortid.MustGenerate())
	resp, err := cl1.SendRequest(req, false)
	assert.Equal(t, resp.Status, transports.StatusPending)
	assert.NoError(t, err)
	<-ctx1.Done()

	// whether receiving completed or delivered depends on the binding
	require.Equal(t, testMsg1, testOutput)

	// 4. verify that the server received it and send a reply
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, inputVal.Load())
	assert.Equal(t, testMsg1, testOutput)

	// 5. Again but wait for the action result
	var result1 string
	err = cl1.InvokeAction(thingID, actionName, testMsg1, &result1)
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, result1)
}

// Warning: this is a bit of a mind bender if you're used to classic consumer->thing interaction.
// This test uses a Thing agent as a client and have it reply to a request from the server.
// The server in this case passes on a message received from a consumer, which is also a client.
func TestInvokeActionFromServerToAgent(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var reqVal atomic.Value
	var replyVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var corrID = "correlation-1"
	var cm *connections.ConnectionManager

	// 1. start the server. register a message handler for receiving an action status
	// async reply from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.

	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn1()
	// server receives agent response
	responseHandler := func(agentID string, resp transports.ResponseMessage) error {
		var responseData string
		// The server receives a response message from the agent
		// (which normally is forwarded to the remote consumer; but not in this test)
		assert.NotEmpty(t, resp.RequestID)
		assert.Equal(t, wot.OpInvokeAction, resp.Operation)

		slog.Info("serverHandler: Received action response from agent",
			"op", resp.Operation,
			"output", resp.Output,
		)
		err := tputils.Decode(resp.Output, &responseData)
		assert.NoError(t, err)

		replyVal.Store(resp.Output)
		cancelFn1()
		return nil
	}
	srv, cancelFn2, cm2 := StartTransportServer(nil, responseHandler, nil)
	cm = cm2
	defer cancelFn2()

	// 2a. connect as an agent
	ag1client := NewAgent(testAgentID1, srv.GetForm)
	token, err := ag1client.ConnectWithPassword(testAgentPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	defer ag1client.Disconnect()

	// an agent receives requests from the server
	ag1client.SetRequestHandler(func(req transports.RequestMessage) transports.ResponseMessage {
		// agent receives action request and returns a result
		slog.Info("Agent receives request", "op", req.Operation)
		assert.Equal(t, testClientID1, req.SenderID)
		reqVal.Store(req.Input)
		return req.CreateResponse(testMsg2, nil)
	})

	// Send the action request from the server to the agent (the agent is connected as a client)
	// and expect result using the request status message sent by the agent.
	time.Sleep(time.Millisecond)
	ag1Server := cm.GetConnectionByClientID(testAgentID1)
	require.NotNil(t, ag1Server)
	req := transports.NewRequestMessage(wot.OpInvokeAction, thingID, actionKey, testMsg1, corrID)
	req.SenderID = testClientID1
	req.RequestID = "rpc-TestInvokeActionFromServerToAgent"
	err = ag1Server.SendRequest(req)
	require.NoError(t, err)

	// wait until the agent has sent a reply
	<-ctx1.Done()

	// if all went well the agent received the request and the server its response
	assert.Equal(t, testMsg1, reqVal.Load())
	assert.Equal(t, testMsg2, replyVal.Load())
}

// TestQueryActions consumer queries the server for actions
// The server receives a QueryAction request and sends a response
func TestQueryActions(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionKey = "action1"

	// 1. start the server. register a request handler for receiving a request
	// from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.
	requestHandler := func(req transports.RequestMessage, replyTo string) transports.ResponseMessage {

		assert.NotNil(t, replyTo)
		assert.NotNil(t, req.RequestID)
		if req.Operation == wot.OpQueryAction {
			// reply a response carrying the queried action response
			actStat := transports.ResponseMessage{
				ThingID:   req.ThingID,
				Name:      req.Name,
				RequestID: req.RequestID,
				Status:    transports.StatusCompleted,
				Output:    testMsg1,
				Received:  req.Created,
				Updated:   time.Now().Format(wot.RFC3339Milli),
			}
			return req.CreateResponse(actStat, nil)

			//replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
		} else if req.Operation == wot.OpQueryAllActions {
			actStat := make([]transports.ResponseMessage, 2)
			actStat[0].ThingID = thingID
			actStat[0].Name = actionKey
			actStat[0].Output = testMsg1
			actStat[1].ThingID = thingID
			actStat[1].Name = actionKey
			actStat[1].RequestID = "requestID-123"
			actStat[1].Status = transports.StatusCompleted
			resp := req.CreateResponse(actStat, nil)
			return resp
			//replyTo.SendResponse(msg.ThingID, msg.Name, actStat, msg.RequestID)
		}
		return req.CreateResponse(nil, errors.New("unexpected response "+req.Operation))
	}

	// 1. start the servers
	srv, cancelFn, _ := StartTransportServer(requestHandler, nil, nil)
	defer cancelFn()

	// 2. connect as a consumer
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// 3. Query action status
	var output transports.ResponseMessage
	err = cl1.Rpc(wot.OpQueryAction, thingID, actionKey, nil, &output)
	require.NoError(t, err)
	require.Equal(t, thingID, output.ThingID)
	require.Equal(t, actionKey, output.Name)

	// 4. Query all actions
	var output2 []transports.ResponseMessage
	err = cl1.Rpc(wot.OpQueryAllActions, thingID, actionKey, nil, &output2)
	require.NoError(t, err)
	require.Equal(t, 2, len(output2))
}
