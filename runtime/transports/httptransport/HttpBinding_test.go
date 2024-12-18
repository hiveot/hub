package httptransport_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/sseclient"
	wssclient "github.com/hiveot/hub/lib/hubclient/wssclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/authn/sessions"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/runtime/transports/httptransport"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"
)

const testPort = 9445

var certBundle = certs.CreateTestCertBundle()
var hostPort = fmt.Sprintf("localhost:%d", testPort)
var testDirFolder = path.Join(os.TempDir(), "test-transport")
var digitwinStorePath = path.Join(testDirFolder, "digitwin.data")
var sm *sessions.SessionManager
var cm *connections.ConnectionManager

var useWSS bool = true // testing using the wss sub-protocol

// ---------
// Dummy sessionAuth for testing the binding
// This implements the authn.IAuthenticator interface.
const clientLoginID = "testlogin"
const clientPassword = "testpass"
const agentLoginID = "agentlogin"
const agentPassword = "agentpass"
const testSessionID = "testsession"

var dummyAuthenticator = &DummyAuthenticator{}

// create a test client as an agent
func newAgentClient(clientID string) (cl hubclient.IAgentClient) {
	if useWSS {
		wssURL := fmt.Sprintf("wss://%s/wss", hostPort)
		cl = wssclient.NewWSSClient(
			wssURL, clientID, nil, certBundle.CaCert, time.Minute)
	} else {
		cl = sseclient.NewHttpSSEClient(
			hostPort, clientID, nil, certBundle.CaCert, time.Minute)
	}
	return cl
}

// create a test client as a consumer
func newConsumerClient(clientID string) (cl hubclient.IConsumerClient) {
	if useWSS {
		wssURL := fmt.Sprintf("wss://%s/wss", hostPort)
		cl = wssclient.NewWSSClient(
			wssURL, clientID, nil, certBundle.CaCert, time.Minute)
	} else {
		cl = sseclient.NewHttpSSEClient(
			hostPort, clientID, nil, certBundle.CaCert, time.Minute)
	}
	return cl
}

// ---------
// startHttpsTransport starts the binding service
// intended to handle the boilerplate
func startHttpsTransport() (
	*httptransport.HttpBinding, *service.DigitwinService, *transports.DummyRouter) {

	// globals in testing
	sm = sessions.NewSessionmanager()
	cm = connections.NewConnectionManager()

	var digitwinRouter = transports.NewDummyRouter(dummyAuthenticator)
	config := httptransport.NewHttpTransportConfig()
	config.Port = testPort

	// start sub-protocol servers
	dtwService, _, err := service.StartDigitwinService(digitwinStorePath, cm)
	if err != nil {
		panic("Failed starting digitwin service:" + err.Error())
	}
	svc, err := httptransport.StartHttpTransport(&config,
		certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator, digitwinRouter, cm)
	if err != nil {
		panic("failed to start binding: " + err.Error())
	}
	dtwService.SetFormsHook(svc.AddTDForms)
	return svc, dtwService, digitwinRouter
}

// create and connect a client for testing
//func createConnectClient(clientID string) *tlsclient.TLSClient {
//	// 2a. create a session for connecting a client
//	// (normally this happens when a session token is issued on authentication)
//	cs, err := sm.AddSession(clientID, "remote addr", "")
//	_ = cs
//	if err != nil {
//		panic("error creating session:" + err.Error())
//	}
//
//	// 2b. connect a client
//	token, sessionID, err := dummyAuthenticator.Login(testLogin, testPassword)
//	_ = sessionID
//	cl := tlsclient.NewTLSClient(hostPort, nil, certBundle.CaCert, time.Second*120, "nocid")
//	cl.SetAuthToken(token)
//	return cl
//}

// TestMain sets logging
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	result := m.Run()
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log("TestStartStop")
	config := httptransport.NewHttpTransportConfig()
	config.Port = testPort
	tb, dtwService, _ := startHttpsTransport()
	assert.NotNil(t, tb)
	assert.NotNil(t, dtwService)
	tb.Stop()
}

func TestLoginRefresh(t *testing.T) {
	t.Log("TestLoginRefresh")
	tb, dtwService, _ := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	cl := newConsumerClient(clientLoginID)
	token, err := cl.ConnectWithPassword(clientPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// refresh should succeed
	newToken, err := cl.RefreshToken(token)
	time.Sleep(time.Millisecond * 30)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)

	// end the session
	cl.Disconnect()

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the authenticator doesn't care.
	token2, err := cl.ConnectWithToken(newToken)
	require.NoError(t, err)
	assert.NotEmpty(t, token2)
	token3, err := cl.RefreshToken(token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, token3)

	// end the session
	cl.Disconnect()
	time.Sleep(time.Millisecond)
}

func TestBadLogin(t *testing.T) {
	t.Log("TestBadLogin")
	tb, dtwService, _ := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	// check if this test still works with a valid login
	cl := newConsumerClient(clientLoginID)
	token, err := cl.ConnectWithPassword(clientPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// failed logins
	token, err = cl.ConnectWithPassword("badpass")
	assert.Error(t, err)
	assert.Empty(t, token)
	token2, err := cl.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token2)
	// close should always succeed
	cl.Disconnect()

	// bad client ID
	cl2 := sseclient.NewHttpSSEClient(hostPort, "badID", nil, certBundle.CaCert, time.Minute)
	token, err = cl2.ConnectWithPassword(clientPassword)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestBadRefresh(t *testing.T) {
	t.Log("TestBadRefresh")
	tb, dtwService, _ := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	cl := newConsumerClient(clientLoginID)

	// set the token
	token, err := cl.ConnectWithToken("badtoken")
	require.Error(t, err)
	assert.Empty(t, token)
	token, err = cl.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token)

	// get a valid token and connect with a bad clientid
	token, err = cl.ConnectWithPassword(clientPassword)
	assert.NoError(t, err)
	validToken, err := cl.RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cl.Disconnect()
	//
	cl2 := newConsumerClient("badclientidlogin")
	defer cl2.Disconnect()
	token, err = cl2.ConnectWithToken(validToken)
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl2.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token)
}

// Test posting an event and action
func TestPostEventAction(t *testing.T) {
	t.Log("TestPostEventAction")
	var rxVal atomic.Value
	var evVal atomic.Value
	var actVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	tb, dtwService, dtwRouter := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	// 2a. create a session for connecting an agent and client
	// (normally this happens when a session token is issued on login)
	sm.NewSession(clientLoginID, "test")
	sm.NewSession(agentLoginID, "test")

	// 2b. connect as an agent and client
	cl1 := newConsumerClient(clientLoginID)
	_, err := cl1.ConnectWithPassword(clientPassword)
	ag1 := newAgentClient(agentLoginID)
	token, err := ag1.ConnectWithPassword(agentPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	cl1.SetMessageHandler(func(ev *hubclient.ThingMessage) {
		// receive result from action
		rxVal.Store(ev.Data)
	})
	cl1.Subscribe("", "")

	// 3. register the dtwRouter for events
	dtwRouter.OnEvent = func(msg *hubclient.ThingMessage) {
		evVal.Store(msg.Data)
	}
	dtwRouter.OnAction = func(msg *hubclient.ThingMessage, replyTo string) (stat hubclient.RequestStatus) {
		actVal.Store(msg.Data)
		stat.Completed(msg, msg.Data, nil)
		return stat
	}
	// 3. publish two events
	err = ag1.PubEvent(thingID, eventKey, testMsg1, "")
	require.NoError(t, err)
	err = ag1.PubEvent(thingID, eventKey, testMsg1, "")
	require.NoError(t, err)

	// 4a. verify that the dtwRouter received it
	time.Sleep(time.Millisecond * 100)
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, evVal.Load())

	// 5. publish an action
	//stat := cl1.InvokeAction(thingID, actionKey, testMsg2, nil, "")
	var reply any
	err = cl1.Rpc(thingID, actionKey, testMsg2, &reply)
	require.NoError(t, err)
	// response must be seen by router and received
	assert.Equal(t, testMsg2, actVal.Load())
	assert.Equal(t, testMsg2, reply)
	ag1.Disconnect()
}

// shortIDs are used in various places
func TestShortID(t *testing.T) {
	total := 100000
	idCounts := make(map[string]int)
	for i := 0; i < total; i++ {
		newID := shortid.MustGenerate()
		idCounts[newID]++
	}
	// check they are unique
	for id, count := range idCounts {
		assert.LessOrEqual(t, 1, count)
		_ = id
	}
	// must have same nr of unique IDs as total
	assert.Equal(t, total, len(idCounts))
}

// Test publish subscribe
func TestPubSub(t *testing.T) {
	t.Log("TestPubSub")
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	tb, dtwService, dtwRouter := startHttpsTransport()
	_ = dtwService

	//svc.PublishEvent(tv.ThingID, tv.Name, tv.Data, tv.CorrelationID)
	defer tb.Stop()

	// 2. connect with a client and agent
	ag1 := newAgentClient(agentLoginID)
	_, err := ag1.ConnectWithPassword(agentPassword)
	require.NoError(t, err)
	defer ag1.Disconnect()
	cl1 := newConsumerClient(clientLoginID)
	_, err = cl1.ConnectWithPassword(clientPassword)
	require.NoError(t, err)
	defer cl1.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// 3. register the dtwRouter for events
	dtwRouter.OnEvent = func(msg *hubclient.ThingMessage) {
		evVal.Store(msg.Data)
	}

	err = cl1.Subscribe(thingID, "")
	require.NoError(t, err)

	// 4. publish an event using the hub client, the server will invoke the message dtwRouter
	// which in turn will publish this to the listeners, including this client.
	err = ag1.PubEvent(thingID, eventKey, testMsg, "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	//
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log("TestReconnect")
	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var dThingID = tdd.MakeDigiTwinThingID(agentID, thingID)

	// 1. start the binding. Set the action handler separately
	svc, dtwService, dtwRouter := startHttpsTransport()
	defer svc.Stop()
	_ = dtwService
	// this test handler receives an action, returns a 'delivered status',
	// and sends a completed status through the sse return channel (SendToClient)

	// reply to requests
	dtwRouter.OnEvent = func(msg *hubclient.ThingMessage) {
	}
	dtwRouter.OnAction = func(msg *hubclient.ThingMessage, replyTo string) (stat hubclient.RequestStatus) {
		// send a completed status update asynchronously
		go func() {
			stat2 := hubclient.RequestStatus{}
			c := svc.GetConnectionByConnectionID(replyTo)
			require.NotNil(t, c)
			stat2.Completed(msg, msg.Data, nil)
			// async completion
			err := c.PublishActionStatus(stat2, agentID)
			assert.NoError(t, err)

		}()
		stat.Delivered(msg)
		return stat
	}

	// 2. connect a service client. Service auth tokens remain valid between sessions.
	cl := newConsumerClient(clientLoginID)
	defer cl.Disconnect()
	token1 := dummyAuthenticator.CreateSessionToken(clientLoginID, "mysession", 1000)
	token2, err := cl.ConnectWithToken(token1)
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	//  Give some time for the SSE connection to be established
	time.Sleep(time.Millisecond * 10)

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	tlsServer := svc.GetHttpServer()
	tlsServer.Stop()
	cm.CloseAll()
	time.Sleep(time.Second)
	tlsServer.Start()

	require.NoError(t, err)

	// give client time to reconnect
	time.Sleep(time.Second * 3)

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs string = "rpc test"
	var rpcResp string
	// this rpc receives a response from the router, not from an agent
	err = cl.Rpc(dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)
}
