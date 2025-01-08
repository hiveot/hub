package runtime_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
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
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	r := startRuntime()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestLoginAsAgent(t *testing.T) {
	const agentID = "agent1"
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	r := startRuntime()
	ag, token := ts.AddConnectAgent(agentID)
	_ = token
	t2, err := ag.RefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, t2)
	// use the refresh token
	t3, err := ag.RefreshToken(t2)
	_ = t3
	require.NoError(t, err)

	ag.Disconnect()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}
func TestLoginAsConsumer(t *testing.T) {
	const clientID = "user1"
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	r := startRuntime()
	cl, token := ts.AddConnectConsumer(clientID, authz.ClientRoleManager)
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

// test many connections from a single consumer and confirm they open close and receive messages properly.
func TestMultiConnectSingleClient(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = int32(100)
	const eventName = "event1"
	var clients = make([]transports.IConsumerConnection, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitafterconnect = time.Millisecond * 10

	// 1: setup: start a runtime and connect N clients
	r := startRuntime()
	ag1, _ := ts.AddConnectAgent(agentID)
	td1 := ts.AddTD(agentID, nil)
	cl1, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)

	onConnection := func(connected bool, err error) {
		if connected {
			connectCount.Add(1)
		} else {
			disConnectCount.Add(1)
		}
	}
	//onRequest := func(req *transports.RequestMessage) transports.ResponseMessage {
	//	messageCount.Add(1)
	//	return req.CreateResponse()
	//}
	onNotification := func(msg transports.NotificationMessage) {
		messageCount.Add(1)
	}
	// 2: connect and subscribe clients and verify
	for range testConnections {
		cl := ts.GetConsumerConnection(clientID1, ts.ConsumerProtocol)
		cl.SetConnectHandler(onConnection)
		cl.SetNotificationHandler(onNotification)
		token, err := cl.ConnectWithToken(token1)
		require.NoError(t, err)
		// allow server to register its connection
		time.Sleep(waitafterconnect)
		err = cl.Subscribe("", "")
		require.NoError(t, err)
		_ = token
		clients = append(clients, cl)
	}
	// connection notification should have been received N times
	time.Sleep(waitafterconnect)
	require.Equal(t, testConnections, connectCount.Load(), "connect count mismatch")

	// 3: agent publishes an event, which should be received N times
	err := ag1.SendNotification(transports.NewNotificationMessage(
		wot.HTOpEvent, td1.ID, eventName, "a value"))
	//err := ag1.PubEvent(td1.ID, eventName, "a value", "message1")
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
	err = ag1.SendNotification(transports.NewNotificationMessage(
		wot.HTOpEvent, td1.ID, eventName, "a value"))
	require.NoError(t, err)
	ag1.Disconnect()

	// zero events should have been received
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int32(0), messageCount.Load(), "still receiving events after disconnect")

	// last, the runtime connection manager should only have no connections
	count, _ := r.CM.GetNrConnections()
	assert.Equal(t, 0, count)
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestActionWithDeliveryConfirmation(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const agentID = "agent1"
	const userID = "user1"
	const actionID = "action-1" // match the test TD action
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"
	var rxMsg transports.RequestMessage

	r := startRuntime()
	defer r.Stop()
	logging.SetLogging("warning", "")
	//slog.SetLogLoggerLevel(slog.LevelWarn)
	ag1, _ := ts.AddConnectAgent(agentID)
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)

	// step 1: agent publishes a TD
	td1 := ts.CreateTestTD(0)
	thingID := td1.ID
	ts.AddTD(agentID, td1)

	// connect the agent and user clients
	defer ag1.Disconnect()
	defer cl1.Disconnect()

	// Agent receives action request which we'll handle here
	agentRequestHandler := func(req transports.RequestMessage) transports.ResponseMessage {
		rxMsg = req
		reply := tputils.DecodeAsString(req.Input, 0) + ".reply"
		// TODO WSS doesn't support the senderID in the message. How important is this?
		// option1: not important - no use-case
		// option2: extend the websocket InvokeAction message format with a SenderID
		//assert.Equal(t, cl1.GetClientID(), msg.SenderID)
		//stat.Failed(msg, fmt.Errorf("failuretest"))
		slog.Info("TestActionWithDeliveryConfirmation: agent1 delivery complete", "correlationID", req.CorrelationID)
		return req.CreateResponse(reply, nil)
	}
	ag1.SetRequestHandler(agentRequestHandler)

	// client sends action to agent and expect a 'delivered' result
	// The RPC method returns an error if no reply is received
	dThingID := td.MakeDigiTwinThingID(agentID, thingID)

	var result string
	err := cl1.Rpc(wot.OpInvokeAction, dThingID, actionID, actionPayload, &result)
	require.NoError(t, err)
	assert.Equal(t, expectedReply, result)
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, actionID, rxMsg.Name)
	assert.Equal(t, vocab.OpInvokeAction, rxMsg.Operation)

}

// Services and agents should auto-reconnect when server is restarted
func TestServiceReconnect(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const agentID = "agent1"
	const userID = "user1"
	var rxMsg atomic.Pointer[transports.RequestMessage]
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"

	r := startRuntime()
	// r is stopped below

	// give server time to start up before connecting
	time.Sleep(time.Millisecond * 10)

	ag1, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()

	// step 1: ensure the thing TD exists
	td1 := ts.CreateTestTD(0)
	thingID := td1.ID
	actionID := "action-1" // match the test TD action
	ts.AddTD(agentID, td1)

	hasAgent := ts.Runtime.CM.GetConnectionByClientID(ag1.GetClientID())
	require.NotNil(t, hasAgent)

	// Agent receives action request which we'll handle here
	ag1.SetRequestHandler(func(msg transports.RequestMessage) transports.ResponseMessage {
		var req string
		rxMsg.Store(&msg)
		_ = tputils.DecodeAsObject(msg.Input, &req)
		output := req + ".reply"
		slog.Info("agent1 delivery complete", "correlationID", msg.CorrelationID)
		return msg.CreateResponse(output, nil)
	})

	// give connection time to be established before stopping the server
	time.Sleep(time.Millisecond * 10)

	// after restarting the server, ag1's connection should automatically be re-established
	// TBD what is the go-sse reconnect algorithm? How to know it triggered?
	t.Log("--- restarting the runtime; 1 existing connection remaining")
	r.Stop()
	time.Sleep(time.Millisecond * 100)

	err := r.Start(&ts.AppEnv)
	require.NoError(t, err)
	defer r.Stop()
	t.Log("--- server restarted; expecting an agent reconnect")

	// wait for the reconnect
	time.Sleep(time.Second * 1)

	hasAgent = ts.Runtime.CM.GetConnectionByClientID(ag1.GetClientID())
	require.NotNil(t, hasAgent)

	cl2, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl2.Disconnect()
	// FIXME: wait for an actual reconnect
	time.Sleep(time.Second * 1)

	// this rpc call succeeds after agent1 has automatically reconnected
	dThingID := td.MakeDigiTwinThingID(agentID, thingID)
	var reply string
	err = cl2.Rpc(wot.OpInvokeAction, dThingID, actionID, &actionPayload, &reply)
	require.NoError(t, err)

	require.NoError(t, err, "auto-reconnect didn't take place")
	rx2 := rxMsg.Load()
	require.NotNil(t, rx2)
	require.Equal(t, expectedReply, string(reply))
}

// test that regular users don't have admin access to authn, authz
func TestAccess(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const clientID = "user1"

	r := startRuntime()
	defer r.Stop()

	hc, token := ts.AddConnectConsumer(clientID, authz.ClientRoleViewer)
	defer hc.Disconnect()
	_ = token

	//f := r.GetForm(wot.OpInvokeAction, hc.GetProtocolType())

	// regulars users should not have authn and authz admin access
	clientProfiles, err := authn.AdminGetProfiles(hc)

	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, clientProfiles)
	//time.Sleep(time.Millisecond * 100)
	// fixme: deadlock in client when two responses are received,
	// first one being unauthorized. [6]
	// second onewith actionstatus=getclientrole output=viewer
	// A: why 2 responses
	// B: why would it deadlock => no-one is reading, no buffer
	// action 1a: Set RPC buffer to 1 so that it can be closed while receiving a followup response
	//              can the second write cause a panic => no, deadlock with 0 buf though
	//              buf of 1 causes response to be lost. should go on as notification.!!
	// action 1b: auto-lock and remove on receive so followup response is a notification.
	role, err := authz.AdminGetClientRole(hc, clientID)
	require.Error(t, err, "regular users should not have access to authz.Admin")
	require.Empty(t, role)
	//time.Sleep(time.Millisecond * 100)
}
