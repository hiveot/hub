package runtime_test

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

var ts *testenv.TestServer

// start the test runtime
func startRuntime() *runtime.Runtime {
	logging.SetLogging("info", "")
	ts = testenv.StartTestServer(true)
	return ts.Runtime
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	r := startRuntime()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const clientID = "user1"

	r := startRuntime()
	cl, token := ts.AddConnectUser(clientID, authz.ClientRoleManager)
	_ = token
	t2, err := cl.RefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, t2)
	// use the refresh token
	t3, err := cl.RefreshToken(t2)
	_ = t3
	require.NoError(t, err)

	cl.Disconnect()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

// test many connections from a single client and confirm they open close and receive messages properly.
func TestMultiConnectSingleClient(t *testing.T) {
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = int32(100)
	const eventName = "event1"
	var clients = make([]hubclient.IConsumerClient, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitafterconnect = time.Millisecond * 10

	// 1: setup: start a runtime and connect N clients
	r := startRuntime()
	ag1, _ := ts.AddConnectAgent(agentID)
	td1 := ts.AddTD(agentID, nil)
	cl1, token1 := ts.AddConnectUser(clientID1, authz.ClientRoleOperator)

	onConnection := func(connected bool, err error) {
		if connected {
			connectCount.Add(1)
		} else {
			disConnectCount.Add(1)
		}
	}
	onMessage := func(msg *hubclient.ThingMessage) {
		messageCount.Add(1)
	}
	// 2: connect and subscribe clients and verify
	for range testConnections {
		serverURL := fmt.Sprintf("https://localhost:%d", ts.Port)
		cl := connect.NewHubClient(serverURL, clientID1, ts.Certs.CaCert)
		cl.SetConnectHandler(onConnection)
		cl.SetMessageHandler(onMessage)
		token, err := cl.ConnectWithToken(token1)
		require.NoError(t, err)
		// allow server to register its connection
		time.Sleep(waitafterconnect)
		err = cl.Subscribe("", "")
		assert.NoError(t, err)
		_ = token
		clients = append(clients, cl)
	}
	// connection notification should have been received N times
	time.Sleep(waitafterconnect)
	require.Equal(t, testConnections, connectCount.Load(), "connect count mismatch")

	// 3: agent publishes an event, which should be received N times
	err := ag1.PubEvent(td1.ID, eventName, "a value", "message1")
	require.NoError(t, err)

	// event should have been received N times
	time.Sleep(time.Millisecond * 100)
	require.Equal(t, testConnections, messageCount.Load(), "missing events")

	// 4: disconnect
	for _, c := range clients {
		c.Disconnect()
	}
	cl1.Disconnect()
	// disconnection notification should have been received N times
	time.Sleep(waitafterconnect)
	require.Equal(t, testConnections, disConnectCount.Load(), "disconnect count mismatch")

	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	err = ag1.PubEvent(td1.ID, eventName, "a value", "message2")
	require.NoError(t, err)
	ag1.Disconnect()

	// zero events should have been received
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int32(0), messageCount.Load(), "still receiving events afer disconnect")

	// last, the runtime connection manager should only have no connections
	count, _ := r.CM.GetNrConnections()
	assert.Equal(t, 0, count)
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestActionWithDeliveryConfirmation(t *testing.T) {
	t.Log("TestActionWithDeliveryConfirmation")
	const agentID = "agent1"
	const userID = "user1"
	const actionID = "action-1" // match the test TD action
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"
	var rxMsg *hubclient.ThingMessage
	var stat3 hubclient.RequestStatus

	r := startRuntime()
	defer r.Stop()
	logging.SetLogging("warning", "")
	//slog.SetLogLoggerLevel(slog.LevelWarn)
	ag1, _ := ts.AddConnectAgent(agentID)
	cl1, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)

	// step 1: agent publishes a TD
	td1 := ts.CreateTestTD(0)
	thingID := td1.ID
	ts.AddTD(agentID, td1)

	// connect the agent and user clients
	defer ag1.Disconnect()
	defer cl1.Disconnect()

	// Agent receives action request which we'll handle here
	ag1.SetRequestHandler(func(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
		rxMsg = msg
		reply := utils.DecodeAsString(msg.Data) + ".reply"
		stat.Completed(msg, reply, nil)
		assert.Equal(t, cl1.GetClientID(), msg.SenderID)
		//stat.Failed(msg, fmt.Errorf("failuretest"))
		slog.Info("TestActionWithDeliveryConfirmation: agent1 delivery complete", "requestID", msg.CorrelationID)
		return stat
	})

	// users receives status updates when sending actions
	deliveryCtx, deliveryCtxComplete := context.WithTimeout(context.Background(), time.Minute*1)
	cl1.SetMessageHandler(func(msg *hubclient.ThingMessage) {
		if msg.Operation == vocab.HTOpUpdateActionStatus {
			// delivery updates are only invoked on for non-rpc actions
			err := utils.DecodeAsObject(msg.Data, &stat3)
			require.NoError(t, err)
			assert.Equal(t, ag1.GetClientID(), msg.SenderID)
			slog.Info(fmt.Sprintf("reply: %s", stat3.Output))
		}
		defer deliveryCtxComplete()
	})
	time.Sleep(time.Millisecond * 10)
	// client sends action to agent and expect a 'delivered' result
	// The RPC method returns an error if no reply is received
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	stat2 := cl1.InvokeAction(dThingID, actionID, actionPayload, nil, "testmsgid")
	require.Empty(t, stat2.Error)

	// wait for delivery completion
	select {
	case <-deliveryCtx.Done():
	}
	time.Sleep(time.Millisecond * 10)

	// verify final result
	require.Equal(t, vocab.RequestCompleted, stat3.Status)
	require.Empty(t, stat3.Error)
	require.NotNil(t, rxMsg)
	assert.Equal(t, expectedReply, stat3.Output)
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, actionID, rxMsg.Name)
	assert.Equal(t, vocab.OpInvokeAction, rxMsg.Operation)

}

// Services and agents should auto-reconnect when server is restarted
func TestServiceReconnect(t *testing.T) {
	t.Log("TestServiceReconnect")
	const agentID = "agent1"
	const userID = "user1"
	var rxMsg atomic.Pointer[*hubclient.ThingMessage]
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"

	r := startRuntime()

	// give server time to start up before connecting
	time.Sleep(time.Millisecond * 10)

	ag1, cl1Token := ts.AddConnectAgent(agentID)
	_ = cl1Token
	defer ag1.Disconnect()

	// step 1: ensure the thing TD exists
	td1 := ts.CreateTestTD(0)
	thingID := td1.ID
	actionID := "action-1" // match the test TD action
	ts.AddTD(agentID, td1)

	// Agent receives action request which we'll handle here
	ag1.SetRequestHandler(func(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
		var req string
		rxMsg.Store(&msg)
		_ = utils.DecodeAsObject(msg.Data, &req)
		stat.Completed(msg, req+".reply", nil)
		slog.Info("agent1 delivery complete", "requestID", msg.CorrelationID)
		return stat
	})

	// give connection time to be established before stopping the server
	time.Sleep(time.Millisecond * 10)

	// after restarting the server, ag1's connection should automatically be re-established
	// TBD what is the go-sse reconnect algorithm? How to know it triggered?
	r.Stop()
	time.Sleep(time.Millisecond * 10)
	err := r.Start(&ts.AppEnv)
	require.NoError(t, err)
	defer r.Stop()

	cl2, _ := ts.AddConnectUser(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()
	// FIXME: detect a reconnect
	time.Sleep(time.Second * 3)

	// this rpc call succeeds after agent1 has automatically reconnected
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)
	var reply string
	err = cl2.Rpc(dThingID, actionID, &actionPayload, &reply)

	require.NoError(t, err, "auto-reconnect didn't take place")
	rx2 := rxMsg.Load()
	require.NotNil(t, rx2)
	require.Equal(t, expectedReply, string(reply))
}

// test that regular users don't have admin access to authn, authz
func TestAccess(t *testing.T) {
	t.Log("TestAccess")
	const clientID = "user1"

	r := startRuntime()
	defer r.Stop()

	hc, token := ts.AddConnectUser(clientID, authz.ClientRoleViewer)
	defer hc.Disconnect()
	_ = token

	// regulars users should not have authn and authz admin access
	prof, err := authn.AdminGetProfiles(hc)
	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, prof)
	//time.Sleep(time.Millisecond * 100)

	role, err := authz.AdminGetClientRole(hc, clientID)
	require.Error(t, err, "regular users should not have access to authz.Admin")
	require.Empty(t, role)
	//time.Sleep(time.Millisecond * 100)
}
