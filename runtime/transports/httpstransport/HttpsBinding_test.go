package httpstransport_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/httpsse"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/transports/httpstransport"
	httpstransport_old "github.com/hiveot/hub/runtime/transports/httpstransport.old"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

const testPort = 9445

var certBundle = certs.CreateTestCertBundle()
var hostPort = fmt.Sprintf("localhost:%d", testPort)

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

// ---------
// startHttpsBinding starts the binding service
// intended to handle the boilerplate
func startHttpsBinding(msgHandler hubclient.MessageHandler) *httpstransport.HttpsTransport {
	config := httpstransport.NewHttpsTransportConfig()
	config.Port = testPort
	svc := httpstransport.NewHttpSSETransport(&config,
		certBundle.ClientKey, certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator)
	err := svc.Start(msgHandler)
	if err != nil {
		panic("failed to start binding: " + err.Error())
	}
	return svc
}

// create and connect a client for testing
func createConnectClient(clientID string) *tlsclient.TLSClient {
	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	sm := sessions.GetSessionManager()
	cs, err := sm.NewSession(clientID, "remote addr", "")
	_ = cs
	if err != nil {
		panic("error creating session:" + err.Error())
	}

	// 2b. connect a client
	token, sessionID, err := dummyAuthenticator.Login(testLogin, testPassword)
	_ = sessionID
	cl := tlsclient.NewTLSClient(hostPort, nil, certBundle.CaCert, time.Second*120)
	cl.SetAuthToken(token)
	return cl
}

// TestMain sets logging
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	result := m.Run()
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log("TestStartStop")
	config := httpstransport_old.NewHttpsTransportConfig()
	config.Port = testPort
	svc := httpstransport_old.NewHttpSSETransport(&config,
		certBundle.ClientKey, certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator,
	)
	err := svc.Start(func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
		stat.Progress = hubclient.DeliveryCompleted
		return stat
	})
	assert.NoError(t, err)
	svc.Stop()
}

func TestLoginRefresh(t *testing.T) {
	t.Log("TestLoginRefresh")
	svc := startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

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
	svc := startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

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
	svc := startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)

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
	cl2 := httpsse.NewHttpSSEClient(hostPort, "badlogin", nil, certBundle.CaCert, time.Minute)
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
	var rxMsg *hubclient.ThingMessage
	var testMsg = "hello world"
	var agentID = "agent1"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	svc := startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			rxMsg = tv
			stat.Completed(tv, testMsg, nil)
			return stat
		})
	defer svc.Stop()

	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	sm := sessions.GetSessionManager()
	cs, err := sm.NewSession(agentID, "remote addr", "test")
	assert.NoError(t, err)
	assert.NotNil(t, cs)

	// 2b. connect a client
	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 3. publish two events
	err = cl.PubEvent(thingID, eventKey, testMsg)
	require.NoError(t, err)
	err = cl.PubEvent(thingID, eventKey, testMsg)
	require.NoError(t, err)

	// 4. verify that the handler received it
	assert.NoError(t, err)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, vocab.MessageTypeEvent, rxMsg.MessageType)
		assert.Equal(t, testMsg, rxMsg.Data.(string))
	}

	// 5. publish an action
	stat := cl.InvokeAction(thingID, actionKey, testMsg)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	assert.NotEmpty(t, stat.Reply)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)
		assert.Equal(t, testMsg, rxMsg.Data.(string))
	}
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
	var rxMsg atomic.Pointer[*hubclient.ThingMessage]
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	var svc *httpstransport_old.HttpsTransport
	svc = startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			// broadcast event to subscribers
			slog.Info("broadcasting event")
			stat = svc.SendEvent(tv)
			return stat
		})
	defer svc.Stop()

	// 2. connect with a client
	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	defer cl.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// 3. register the handler for events
	cl.SetMessageHandler(
		func(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			rxMsg.Store(&msg)
			return stat.Completed(msg, nil, nil)
		})
	err = cl.Subscribe(thingID, "")
	require.NoError(t, err)

	// 4. publish an event using the hub client, the server will invoke the message handler
	// which in turn will publish this to the listeners over sse, including this client.
	err = cl.PubEvent(thingID, eventKey, testMsg)
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	//
	rxMsg2 := rxMsg.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, thingID, (*rxMsg2).ThingID)
	assert.Equal(t, testLogin, (*rxMsg2).SenderID)
	assert.Equal(t, eventKey, (*rxMsg2).Name)
}

// Restarting the server should invalidate sessions
func TestRestart(t *testing.T) {
	t.Log("TestRestart")
	var rxMsg *hubclient.ThingMessage
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the binding
	svc := startHttpsBinding(
		func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			return stat
		})

	// 2. connect a service client
	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	token, err := cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// restart the server. This should invalidate session auth
	t.Log("--- Stopping the server ---")
	svc.Stop()
	err = svc.Start(func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
		rxMsg = tv
		stat.Completed(tv, testMsg, nil)
		return stat
	})
	t.Log("--- Restarted the server ---")
	require.NoError(t, err)
	defer svc.Stop()

	// 3. publish an event should fail without a login
	err = cl.PubEvent(thingID, eventKey, testMsg)
	require.Error(t, err)

	require.Nil(t, rxMsg)
	//assert.Equal(t, eventKey, rxMsg.Name)
	//assert.Equal(t, thingID, rxMsg.ThingID)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log("TestReconnect")
	var thingID = "thing1"
	var actionKey = "action1"
	var actionHandler func(*hubclient.ThingMessage) hubclient.DeliveryStatus

	// 1. start the binding. Set the action handler separately
	svc := startHttpsBinding(
		func(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
			if actionHandler != nil {
				return actionHandler(msg)
			}
			stat.Failed(msg, fmt.Errorf("No test action handler"))
			return stat
		})
	defer svc.Stop()

	// this test handler receives an action, returns a 'delivered status',
	// and sends a completed status through the sse return channel (SendToClient)
	actionHandler = func(tv *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
		stat.Progress = hubclient.DeliveredToAgent
		if tv.MessageType == vocab.MessageTypeEvent {
			// ignore events
			return stat
		}
		// send a delivery status update asynchronously which uses the SSE return channel
		go func() {
			var stat2 hubclient.DeliveryStatus
			stat2.Completed(tv, tv.Data, nil)
			tm2 := hubclient.NewThingMessage(
				vocab.MessageTypeEvent, tv.SenderID, vocab.EventNameDeliveryUpdate, stat2, thingID)

			svc.SendToClient(tv.SenderID, tm2)
		}()
		stat.Progress = hubclient.DeliveryApplied
		return stat
	}

	// 2. connect a service client. Service auth tokens remain valid between sessions.
	cl := httpsse.NewHttpSSEClient(hostPort, testLogin, nil, certBundle.CaCert, time.Minute)
	defer cl.Disconnect()
	token, err := cl.ConnectWithToken(serviceToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	//cl.PubEvent("dummything", "dummyKey", nil)

	//  Give some time for the SSE connection to be re-established
	time.Sleep(time.Second * 1)

	// 3. restart the server.
	t.Log("--- restarting the server ---")
	svc.Stop()
	time.Sleep(time.Millisecond * 10)
	err = svc.Start(actionHandler)
	require.NoError(t, err)
	t.Log("--- server restarted ---")

	// give client time to reconnect
	time.Sleep(time.Second * 3)
	// publish event to rekindle the connection
	cl.PubEvent("dummything", "dummyKey", "")
	// 4. The SSE return channel should reconnect automatically
	var rpcArgs string = "rpc test"
	var rpcResp string
	err = cl.Rpc(thingID, actionKey, &rpcArgs, &rpcResp)
	assert.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

}
