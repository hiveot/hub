package runtime_test

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/runtime/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEnv *testenv.TestEnv

// start the test runtime
func startRuntime() *pkg.HubRuntime {
	utils.SetLogging("info", "")
	testEnv = testenv.NewTestEnv(true)
	// testEnv.StartTestServer("")
	rt, f, err := pkg.NewHubRuntime(testEnv.Env)
	_ = f
	_ = err
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
	rcThing, ccag, token := testEnv.NewRCThing(agentID, nil) //AddConnectAgent(agentID)
	defer rcThing.Stop()
	defer ccag.Close()
	authCl := authn_client.NewAuthnUserClient("", rcThing)
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
	co1, cc1, token := testEnv.NewConnectedConsumer(clientID, api.ClientRoleManager)
	defer co1.Stop()
	defer cc1.Stop()
	authCl := authn_client.NewAuthnUserClient("", co1)
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
	const thing1ID = "thing1"
	const testConnections = int32(100)
	const eventName = "event1"
	var clients = make([]api.ITransportClient, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitafterconnect = time.Millisecond * 10

	// 1: setup: start a runtime and connect N clients
	r := startRuntime()
	rcThing, ccag, _ := testEnv.NewRCThing(agentID, nil)
	defer ccag.Close()

	// td1 := testEnv.AddTD(agentID, nil)
	co1, cc1, token1 := testEnv.NewConnectedConsumer(clientID1, api.ClientRoleOperator)
	_ = co1

	onConnection := func(newStatus api.ConnectionStatus, c api.ITransportClient) {
		if newStatus == api.StatusConnected {
			connectCount.Add(1)
		} else if newStatus == api.StatusClosed || newStatus == api.StatusLost {
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
		co, cc, _ := testEnv.NewConnectedConsumer(clientID1, api.ClientRoleOperator)
		cc.SetConnectHandler(onConnection)
		co.SetNotificationHook(onNotification)
		err := cc.SetAuthToken(clientID1, token1, wot.SecSchemeBearer)
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
	rcThing.PubEvent(thing1ID, eventName, "a value")
	//err := ag1.PubEvent(td1.ID, eventName, "a value", "message1")
	// require.NoError(t, err)

	// event should have been received N times (in debug mode this can be rather slow)
	time.Sleep(time.Millisecond * 500)
	require.Equal(t, testConnections, messageCount.Load(), "missing events")

	// 4: disconnect
	for _, c := range clients {
		c.Close()
	}
	cc1.Stop()
	// disconnection notification should have been received N times
	time.Sleep(waitafterconnect * 3)
	require.Equal(t, testConnections, disConnectCount.Load(), "disconnect count mismatch")

	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	rcThing.PubEvent(thing1ID, eventName, "a value")
	// require.NoError(t, err)
	rcThing.Stop()

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

	cc1, token := testEnv.NewTestClient(clientID, api.ClientRoleAdmin)
	cc1.Connect()
	defer cc1.Close()
	_ = token

	//f := r.GetForm(wot.OpInvokeAction, hc.GetProtocolType())

	// regulars users should not have authn and authz admin access
	authnAdmin := authn_client.NewAuthnAdminClient("", cc1)
	clientProfiles, err := authnAdmin.GetProfiles()

	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, clientProfiles)
	//time.Sleep(time.Millisecond * 100)
	prof, err := authnAdmin.GetProfile(clientID)
	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, prof.Role)
	//time.Sleep(time.Millisecond * 100)
}
