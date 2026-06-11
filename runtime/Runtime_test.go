package runtime_test

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEnv *testenv.TestEnv

// start the test runtime
func startRuntime() *runtime.Runtime {
	utils.SetLogging("info", "")
	testEnv = testenv.NewTestEnv()
	testEnv.StartTestServer(testenv.DefaultProtocol)
	rt := runtime.NewRuntime()
	return rt
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	r := startRuntime()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestLoginAsAgent(t *testing.T) {
	const agentID = "agent1"
	t.Logf("---%s---\n", t.Name())

	r := startRuntime()
	agent, ccag, token := testEnv.NewRCAgent(agentID, nil) //AddConnectAgent(agentID)
	defer agent.Stop()
	defer ccag.Close()
	authCl := authnpkg.NewAuthnUserClient(agent)
	t2, err := authCl.RefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, t2)

	// use the refresh token
	t3, err := authCl.RefreshToken(t2)
	_ = t3
	require.NoError(t, err)

	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}
func TestLoginAsConsumer(t *testing.T) {
	const clientID = "user1"
	t.Logf("---%s---\n", t.Name())

	r := startRuntime()
	co1, cc, token := testEnv.NewConnectedConsumer(clientID, authn.ClientRoleManager, false)
	defer co1.Stop()
	defer cc.Close()
	authCl := authnpkg.NewAuthnUserClient(cc)
	t2, err := authCl.RefreshToken(token)

	require.NoError(t, err)
	assert.NotEmpty(t, t2)

	// use the refresh token
	t3, err := authCl.RefreshToken(t2)
	require.NoError(t, err)
	require.NotEmpty(t, t3)

	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

// test many connections from a single consumer and confirm they open close and receive messages properly.
func TestMultiConnectSingleClient(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = int32(100)
	const eventName = "event1"
	var clients = make([]transport.ITransportClient, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitafterconnect = time.Millisecond * 10

	// 1: setup: start a runtime and connect N clients
	r := startRuntime()
	ag1, ccag, _ := testEnv.NewRCAgent(agentID, nil)
	defer ccag.Close()
	td1 := testEnv.AddTD(agentID, nil)
	cl1, _, token1 := testEnv.NewConnectedConsumer(clientID1, authn.ClientRoleOperator, false)

	onConnection := func(newStatus transport.ConnectionStatus, c transport.ITransportClient) {
		if newStatus == transport.StatusConnected {
			connectCount.Add(1)
		} else if newStatus == transport.StatusClosed || newStatus == transport.StatusLost {
			disConnectCount.Add(1)
		}
	}
	//onRequest := func(req *transports.RequestMessage) transports.ResponseMessage {
	//	messageCount.Add(1)
	//	return req.CreateResponse()
	//}
	onNotification := func(notif *msg.NotificationMessage) {
		messageCount.Add(1)
	}
	// 2: connect and subscribe clients and verify
	for range testConnections {
		co, cc, _ := testEnv.NewConnectedConsumer(clientID1, authn.ClientRoleOperator, false)
		cc.SetConnectHandler(onConnection)
		co.SetNotificationHook(onNotification)
		err := cc.AuthenticateWithToken(clientID1, token1)
		require.NoError(t, err)
		// allow server to register its connection
		time.Sleep(waitafterconnect)
		err = co.Subscribe("", "")
		require.NoError(t, err)
		clients = append(clients, cc)
	}
	// connection notification should have been received N times
	time.Sleep(waitafterconnect)
	require.Equal(t, testConnections, connectCount.Load(), "connect count mismatch")
	//require.Equal(t, testConnections, ts.Runtime.TransportsMgr.GetNrConnections(), "ts connect count mismatch")

	// 3: agent publishes an event, which should be received N times
	ag1.PubEvent(td1.ID, eventName, "a value")
	//err := ag1.PubEvent(td1.ID, eventName, "a value", "message1")
	// require.NoError(t, err)

	// event should have been received N times (in debug mode this can be rather slow)
	time.Sleep(time.Millisecond * 500)
	require.Equal(t, testConnections, messageCount.Load(), "missing events")

	// 4: disconnect
	for _, c := range clients {
		c.Close()
	}
	cl1.Stop()
	// disconnection notification should have been received N times
	time.Sleep(waitafterconnect * 3)
	require.Equal(t, testConnections, disConnectCount.Load(), "disconnect count mismatch")

	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	ag1.PubEvent(td1.ID, eventName, "a value")
	// require.NoError(t, err)
	ag1.Stop()

	// zero events should have been received
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, int32(0), messageCount.Load(), "still receiving events after disconnect")

	// last, the runtime connection manager should only have no connections
	//count, _ := r.TransportsMgr.GetNrConnections()
	//assert.Equal(t, 0, count)
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

// test that regular users don't have admin access to authn, authz
func TestAccess(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID = "user1"

	r := startRuntime()
	defer r.Stop()

	cc1, token := testEnv.NewConnectedClient(clientID, authn.ClientRoleAdmin)
	defer cc1.Close()
	_ = token

	//f := r.GetForm(td.OpInvokeAction, hc.GetProtocolType())

	// regulars users should not have authn and authz admin access
	authnAdmin := authnpkg.NewAuthnAdminClient(cc1)
	clientProfiles, err := authnAdmin.GetProfiles()

	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, clientProfiles)
	//time.Sleep(time.Millisecond * 100)
	prof, err := authnAdmin.GetClientProfile(clientID)
	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, prof.Role)
	//time.Sleep(time.Millisecond * 100)
}
