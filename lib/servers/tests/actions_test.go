package tests

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hub/lib/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
)

// TestInvokeActionFromConsumerToServer: classic 'consumer talks to the server'
// as if it is a Thing. In this test the server replies.
// (routing is not part of this package)
func TestInvokeActionFromConsumerToServer(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	//var outputVal atomic.Value
	var testOutput string
	var testActionStatus messaging.ActionStatus

	var inputVal atomic.Value
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionName = "action1"

	// the server will receive the action request and return an immediate result
	requestHandler := func(req *messaging.RequestMessage, replyTo messaging.IConnection) *messaging.ResponseMessage {
		if req.Operation == wot.OpInvokeAction {
			inputVal.Store(req.Input)
			// CreateResponse returns ActionStatus
			return req.CreateResponse(req.Input, nil)
		}
		assert.Fail(t, "Not expecting this")
		return req.CreateResponse(nil, errors.New("unexpected request"))
	}
	// 1. start the servers
	srv, cancelFn := StartTransportServer(nil, requestHandler, nil)
	_ = srv
	defer cancelFn()

	// 2. connect a client
	cc1, cl1, token := NewConsumer(testClientID1)
	defer cc1.Disconnect()
	require.NotEmpty(t, token)
	ctx1, release1 := context.WithTimeout(context.Background(), time.Minute)
	defer release1()

	// since there is no waiting for a response when sending the request, the
	// client should receive an action/request response via the response callback
	cl1.SetResponseHandler(func(resp *messaging.ResponseMessage) error {
		slog.Info("testOutput was updated asynchronously via the message handler")
		// response should be an ActionStatus object
		err2 := utils.Decode(resp.Value, &testActionStatus)
		if assert.NoError(t, err2) {
			err2 = utils.DecodeAsObject(testActionStatus.Output, &testOutput)
			assert.NoError(t, err2)
		}
		release1()
		return err2
	})

	// 3. invoke the action without waiting for a result
	// the response handler above will receive the result
	// testOutput can be updated as an immediate result or via the callback message handler
	req := messaging.NewRequestMessage(wot.OpInvokeAction, thingID, actionName, testMsg1, shortid.MustGenerate())
	resp, err := cl1.SendRequest(req, false)
	// waitForCompletion is false so no response yet
	require.NoError(t, err)
	assert.Nil(t, resp)
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
	t.Logf("---%s---\n", t.Name())
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
	// server receives agent response
	responseHandler := func(resp *messaging.ResponseMessage) error {
		var responseData string
		// The server receives a response message from the agent
		// (which normally is forwarded to the remote consumer; but not in this test)
		assert.NotEmpty(t, resp.CorrelationID)
		assert.Equal(t, wot.OpInvokeAction, resp.Operation)

		slog.Info("serverHandler: Received action response from agent",
			"op", resp.Operation,
			"output", resp.Value,
		)
		err := resp.Decode(&responseData)
		assert.NoError(t, err)

		replyVal.Store(responseData)
		cancelFn1()
		return nil
	}
	srv, cancelFn2 := StartTransportServer(nil, nil, responseHandler)
	_ = srv
	defer cancelFn2()

	// 2a. connect as an agent
	cc1, ag1client, token := NewAgent(testAgentID1)
	require.NotEmpty(t, token)
	defer cc1.Disconnect()

	// an agent receives requests from the server
	ag1client.SetRequestHandler(func(req *messaging.RequestMessage, replyTo messaging.IConnection) *messaging.ResponseMessage {
		// agent receives action request and returns a result
		slog.Info("Agent receives request", "op", req.Operation)
		assert.Equal(t, testClientID1, req.SenderID)
		reqVal.Store(req.Input)
		go func() {
			time.Sleep(time.Millisecond)
			// separately send a completed response
			resp := req.CreateResponse(testMsg2, nil)
			_ = replyTo.SendResponse(resp)
		}()
		// the response is sent asynchronously
		return nil
	})

	// Send the action request from the server to the agent (the agent is connected as a client)
	// and expect result using the request status message sent by the agent.
	time.Sleep(time.Millisecond)
	ag1Server := srv.GetConnectionByClientID(testAgentID1)
	require.NotNil(t, ag1Server)
	req := messaging.NewRequestMessage(wot.OpInvokeAction, thingID, actionKey, testMsg1, corrID)
	req.SenderID = testClientID1
	req.CorrelationID = "rpc-TestInvokeActionFromServerToAgent"
	err := ag1Server.SendRequest(req)
	require.NoError(t, err)

	// wait until the agent has sent a reply
	<-ctx1.Done()
	time.Sleep(time.Millisecond * 10)

	// if all went well the agent received the request and the server its response
	assert.Equal(t, testMsg1, reqVal.Load())
	assert.Equal(t, testMsg2, replyVal.Load())
}

// TestQueryActions consumer queries the server for actions
// The server receives a QueryAction request and sends a response
func TestQueryActions(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var testMsg1 = "hello world 1"
	var thingID = "dtw:thing1"
	var actionKey = "action1"
	var actionID = "correlationID-123"

	// 1. start the server. register a request handler for receiving a request
	// from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.
	requestHandler := func(req *messaging.RequestMessage, replyTo messaging.IConnection) *messaging.ResponseMessage {

		assert.NotNil(t, replyTo)
		assert.NotNil(t, req.CorrelationID)
		switch req.Operation {
		case wot.OpQueryAction:
			// reply a response carrying the queried action status
			actStat := messaging.ActionStatus{
				ThingID:       req.ThingID,
				Name:          req.Name,
				ActionID:      actionID,
				Output:        testMsg1,
				State:         messaging.StatusCompleted,
				TimeRequested: req.Created,
				TimeUpdated:   utils.FormatNowUTCMilli(),
			}

			return req.CreateResponse(actStat, nil)

			//replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.CorrelationID)
		case wot.OpQueryAllActions:
			// include an error status to ensure encode/decode of an error status works
			actStat := map[string]messaging.ActionStatus{
				actionKey: {
					ThingID:       req.ThingID,
					Name:          actionKey,
					ActionID:      actionID,
					Output:        testMsg1,
					State:         messaging.StatusCompleted,
					TimeRequested: req.Created,
					TimeUpdated:   utils.FormatNowUTCMilli(),
				},
				"action-2": {
					ThingID:  req.ThingID,
					Name:     "action-2",
					ActionID: actionID,
					Error: &messaging.ErrorValue{
						Status: http.StatusBadRequest,
						Type:   "http://testerror/",
						Title:  "Testing error",
						Detail: "test error detail",
					},
					State:         messaging.StatusFailed,
					TimeRequested: utils.FormatNowUTCMilli(),
					TimeUpdated:   utils.FormatNowUTCMilli(),
				},
				"action-3": {
					ThingID:       req.ThingID,
					Name:          "action-3",
					ActionID:      actionID,
					Output:        "other output",
					State:         messaging.StatusCompleted,
					TimeRequested: utils.FormatNowUTCMilli(),
					TimeUpdated:   utils.FormatNowUTCMilli(),
				}}
			resp := req.CreateResponse(actStat, nil)
			return resp
		}
		return req.CreateResponse(nil, errors.New("unexpected response "+req.Operation))
	}

	// 1. start the servers
	srv, cancelFn := StartTransportServer(nil, requestHandler, nil)
	_ = srv
	defer cancelFn()

	// 2. connect as a consumer
	cc1, cl1, _ := NewConsumer(testClientID1)
	defer cc1.Disconnect()

	// 3. Query action status
	var status messaging.ActionStatus
	err := cl1.Rpc(wot.OpQueryAction, thingID, actionKey, nil, &status)
	require.NoError(t, err)
	require.Equal(t, thingID, status.ThingID)
	require.Equal(t, actionKey, status.Name)

	// 4. Query all actions
	var statusMap map[string]messaging.ActionStatus
	err = cl1.Rpc(wot.OpQueryAllActions, thingID, actionKey, nil, &statusMap)
	require.NoError(t, err)
	require.Equal(t, 3, len(statusMap))
}
