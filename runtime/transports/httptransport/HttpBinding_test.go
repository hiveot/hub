package httptransport_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/httpsse"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/digitwin/service"
	sessions2 "github.com/hiveot/hub/runtime/sessions"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/runtime/transports/httptransport"
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
var sm *sessions2.SessionManager
var cm *sessions2.ConnectionManager

// ---------
// Dummy sessionAuth for testing the binding
// This implements the authn.IAuthenticator interface.
const testLogin = "testlogin"
const testPassword = "testpass"
const userToken = "usertoken"
const serviceToken = "servicetoken"
const testSessionID = "testSession"

type DummyAuthenticator struct{}

func (d *DummyAuthenticator) CreateSessionToken(
	clientID, sessionID string, validitySec int) (token string) {
	if sessionID != "" {
		return userToken
	}
	return serviceToken
}

func (d *DummyAuthenticator) Login(
	clientID string, password string) (token string, sessionID string, err error) {

	if password == testPassword && clientID == testLogin {
		return userToken, testSessionID, err
	}
	return token, sessionID, fmt.Errorf("Invalid login")
}
func (d *DummyAuthenticator) ValidatePassword(clientID string, password string) (err error) {
	return nil
}

func (d *DummyAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, err error) {
	return oldToken, nil
}

func (d *DummyAuthenticator) ValidateToken(
	token string) (clientID string, sessionID string, err error) {

	if token == userToken {
		return testLogin, testSessionID, nil
	} else if token == serviceToken {
		return testLogin, "", nil
	}
	err = fmt.Errorf("invalid login")
	return clientID, sessionID, err
}

var dummyAuthenticator = &DummyAuthenticator{}

// create a test client as an agent
func newAgentClient(clientID string) hubclient.IAgentClient {
	cl := httpsse.NewHttpSSEClient(hostPort, clientID, nil, certBundle.CaCert, time.Minute)
	return cl
}

// create a test client as a consumer
func newConsumerClient(clientID string) hubclient.IConsumerClient {
	cl := httpsse.NewHttpSSEClient(hostPort, clientID, nil, certBundle.CaCert, time.Minute)
	return cl
}

// ---------
// startHttpsTransport starts the binding service
// intended to handle the boilerplate
func startHttpsTransport() (
	*httptransport.HttpBinding, *service.DigitwinService, *transports.DummyRouter) {

	// globals in testing
	sm = sessions2.NewSessionmanager()
	cm = sessions2.NewConnectionManager()

	var hubRouter = &transports.DummyRouter{}
	config := httptransport.NewHttpTransportConfig()
	config.Port = testPort

	// start sub-protocol servers
	dtwService, _, err := service.StartDigitwinService(digitwinStorePath, cm)
	if err != nil {
		panic("Failed starting digitwin service:" + err.Error())
	}
	svc, err := httptransport.StartHttpTransport(&config,
		certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator, hubRouter, dtwService, cm, sm)
	if err != nil {
		panic("failed to start binding: " + err.Error())
	}
	dtwService.SetFormsHook(svc.AddTDForms)
	return svc, dtwService, hubRouter
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

	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	token, err := cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// refresh should succeed
	newToken, err := cl.RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)

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
	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	//cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	token, err := cl.ConnectWithPassword(testPassword)
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
	cl2 := httpsse.NewHttpSSEClient(hostPort, "badID", nil, certBundle.CaCert, time.Minute)
	token, err = cl2.ConnectWithPassword(testPassword)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestBadRefresh(t *testing.T) {
	t.Log("TestBadRefresh")
	tb, dtwService, _ := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	cl := newConsumerClient(testLogin)

	// set the token
	token, err := cl.ConnectWithToken("badtoken")
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token)

	// get a valid token and connect with a bad clientid
	token, err = cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	validToken, err := cl.RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cl.Disconnect()
	//
	cl2 := newConsumerClient("badlogin")
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
	var evVal atomic.Value
	var actVal atomic.Value
	var testMsg = "hello world"
	var agentID = "agent1"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	tb, dtwService, handler := startHttpsTransport()
	_ = dtwService
	defer tb.Stop()

	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	cs, err := sm.AddSession(agentID, "remote addr", "test")
	assert.NoError(t, err)
	assert.NotNil(t, cs)

	// 2b. connect as an agent
	cl := newAgentClient(testLogin)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 3. register the handler for events
	handler.OnEvent = func(agentID, thingID, name string, val any, msgID string) {
		evVal.Store(val)
	}
	handler.OnAction = func(agentID, thingID, name string, val any, msgID string, cid string) any {
		actVal.Store(val)
		return val
	}
	// 3. publish two events
	err = cl.PubEvent(thingID, eventKey, testMsg, "")
	require.NoError(t, err)
	err = cl.PubEvent(thingID, eventKey, testMsg, "")
	require.NoError(t, err)

	// 4. verify that the handler received it
	assert.NoError(t, err)
	assert.Equal(t, testMsg, evVal.Load())

	// 5. publish an action
	stat := cl.InvokeAction(thingID, actionKey, testMsg, "")
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	assert.NotEmpty(t, stat.Reply)
	assert.Equal(t, testMsg, actVal.Load())
	cl.Disconnect()
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

// Test publish subscribe using sse
func TestPubSubSSE(t *testing.T) {
	t.Log("TestPubSubSSE")
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	tb, dtwService, handler := startHttpsTransport()
	_ = dtwService

	//svc.PublishEvent(tv.ThingID, tv.Name, tv.Data, tv.MessageID)
	defer tb.Stop()

	// 2. connect with a client
	cl := newAgentClient(testLogin)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	defer cl.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// 3. register the handler for events
	handler.OnEvent = func(agentID, thingID, name string, val any, msgID string) {
		evVal.Store(val)
	}

	err = cl.Subscribe(thingID, "")
	require.NoError(t, err)

	// 4. publish an event using the hub client, the server will invoke the message handler
	// which in turn will publish this to the listeners over sse, including this client.
	err = cl.PubEvent(thingID, eventKey, testMsg, "")
	assert.NoError(t, err)
	//time.Sleep(time.Millisecond * 10)
	//
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}

// Restarting the server should invalidate sessions
func TestRestart(t *testing.T) {
	t.Log("TestRestart")
	//var rxMsg *hubclient.ThingMessage
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the binding
	svc, dtwService, _ := startHttpsTransport()
	_ = dtwService
	// 2. connect as an agent
	cl := newAgentClient(testLogin)
	token, err := cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// restart the server. This should invalidate session auth
	t.Log("--- Stopping the server ---")
	cm.CloseAll()
	svc.Stop()
	svc, _, _ = startHttpsTransport()
	dtwService.SetFormsHook(svc.AddTDForms)

	t.Log("--- Restarted the server ---")
	require.NoError(t, err)
	defer svc.Stop()

	// 3. even though the login creds are still valid in the dummy authenticator,
	//  the associated session should have been removed.
	err = cl.PubEvent(thingID, eventKey, testMsg, "")
	require.Error(t, err)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log("TestReconnect")
	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	//var dThingID = tdd.MakeDigiTwinThingID(agentID, thingID)

	// 1. start the binding. Set the action handler separately
	svc, dtwService, dummyRouter := startHttpsTransport()
	defer svc.Stop()
	_ = dtwService
	// this test handler receives an action, returns a 'delivered status',
	// and sends a completed status through the sse return channel (SendToClient)

	// 2. connect a service client. Service auth tokens remain valid between sessions.
	cl := newAgentClient(testLogin)
	defer cl.Disconnect()
	token, err := cl.ConnectWithToken(serviceToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	//  Give some time for the SSE connection to be re-established
	time.Sleep(time.Second * 1)

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- disconnecting all clients ---")
	cm.CloseAll()

	// reply to requests after a restart
	dummyRouter.OnEvent = func(agentID, thingID, name string, val any, msgID string) {
		return
	}
	dummyRouter.OnAction = func(agentID, thingID, name string, val any, msgID string, cid string) any {
		// send a delivery status update asynchronously which uses the SSE return channel
		go func() {
			stat := hubclient.ActionProgress{
				MessageID: msgID,
				Progress:  vocab.ProgressStatusCompleted,
				Reply:     val,
			}
			c := svc.GetConnectionByCID(cid)
			require.NotNil(t, c)
			err = c.PublishActionProgress(stat, agentID)
			assert.NoError(t, err)
		}()
		return nil
	}
	require.NoError(t, err)

	// give client time to reconnect
	time.Sleep(time.Second * 3)
	// publish event to rekindle the SSE connection
	cl.PubEvent("dummything", "dummyKey", "", "")

	// 4. The SSE return channel should reconnect automatically
	// An RPC call is the ultimate test
	//var rpcArgs string = "rpc test"
	//var rpcResp string
	//err = cl.Rpc(dThingID, actionKey, &rpcArgs, &rpcResp)
	//assert.NoError(t, err)
	//assert.Equal(t, rpcArgs, rpcResp)

}
