package httpstransport_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/httpclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/httpstransport"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

const testPort = 8444

var certBundle = certs.CreateTestCertBundle()
var hostPort = fmt.Sprintf("localhost:%d", testPort)

// ---------
// Dummy sessionAuth for testing the binding
// This implements the authn.IAuthenticator interface.
const testLogin = "testlogin"
const testPassword = "testpass"
const testToken = "testtoken"

type DummyAuthenticator struct{}

func (d *DummyAuthenticator) Login(clientID string, password string, sessionID string) (token string, sid string, err error) {
	if sessionID == "" {
		//uid, _ := uuid.NewUUID()
		//sessionID = uid.String()
		sessionID = "testsession"
	}
	if password == testPassword && clientID == testLogin {
		return testToken, sessionID, nil
	}
	return "", "", fmt.Errorf("Invalid login")
}
func (d *DummyAuthenticator) CreateSessionToken(clientID, sessionID string, validitySec int) (token string) {
	return testToken
}

func (d *DummyAuthenticator) RefreshToken(clientID string, oldToken string, validitySec int) (newToken string, err error) {
	return testToken, nil
}

func (d *DummyAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {
	if token != testToken {
		return "", "", fmt.Errorf("invalid login")
	}
	return testLogin, "testsession", nil
}

var dummyAuthenticator = &DummyAuthenticator{}

// ---------
// startHttpsBinding starts the binding service
// intended to handle the boilerplate
func startHttpsBinding(msgHandler api.MessageHandler) *httpstransport.HttpsTransport {
	config := httpstransport.NewHttpsTransportConfig()
	config.Port = testPort
	svc := httpstransport.NewHttpsBinding(&config,
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
	sessionToken, sid, err := dummyAuthenticator.Login(testLogin, testPassword, "")
	_ = sid

	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	cl.ConnectWithToken(clientID, sessionToken)
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
	config := httpstransport.NewHttpsTransportConfig()
	config.Port = testPort
	svc := httpstransport.NewHttpsBinding(&config,
		certBundle.ClientKey, certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator,
	)
	err := svc.Start(func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
		stat.Status = api.DeliveryCompleted
		return stat
	})
	assert.NoError(t, err)
	svc.Stop()
}

func TestLoginRefresh(t *testing.T) {
	t.Log("TestLoginRefresh")
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

	// the dummy authenticator accepts only the testLogin and testPassword
	cl := httpclient.NewHttpSSEClient(hostPort, testLogin, certBundle.CaCert)
	token, err := cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// refresh should succeed
	token, err = cl.RefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// end the session
	cl.Disconnect()

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the authenticator doesn't care.
	token, err = cl.ConnectWithJWT(token)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	token2, err := cl.RefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token2)

	// end the session
	cl.Disconnect()
	time.Sleep(time.Millisecond)
}

func TestBadLogin(t *testing.T) {
	t.Log("TestBadLogin")
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

	// check if this test still works with a valid login
	cl := httpclient.NewHttpSSEClient(hostPort, testLogin, certBundle.CaCert)
	//cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	token, err := cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// failed logins
	token, err = cl.ConnectWithPassword("badpass")
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl.RefreshToken()
	assert.Error(t, err)
	assert.Empty(t, token)
	// close should always succeed
	cl.Disconnect()

	// bad client ID
	cl2 := httpclient.NewHttpSSEClient(hostPort, "badID", certBundle.CaCert)
	token, err = cl2.ConnectWithPassword(testPassword)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestBadRefresh(t *testing.T) {
	t.Log("TestBadRefresh")
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
			assert.Fail(t, "should not get here")
			return stat
		})
	defer svc.Stop()

	cl := httpclient.NewHttpSSEClient(hostPort, testLogin, certBundle.CaCert)

	// set the token
	token, err := cl.ConnectWithJWT("badtoken")
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl.RefreshToken()
	assert.Error(t, err)
	assert.Empty(t, token)

	// get a valid token and connect with a bad clientid
	token, err = cl.ConnectWithPassword(testPassword)
	assert.NoError(t, err)
	validToken, err := cl.RefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cl.Disconnect()
	//
	cl2 := httpclient.NewHttpSSEClient(hostPort, "badlogin", certBundle.CaCert)
	defer cl2.Disconnect()
	token, err = cl2.ConnectWithJWT(validToken)
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl2.RefreshToken()
	assert.Error(t, err)
	assert.Empty(t, token)
}

// Test posting an event
func TestPostEventAction(t *testing.T) {
	t.Log("TestPostEventAction")
	var rxMsg *things.ThingMessage
	var testMsg = "hello world"
	var agentID = "agent1"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
			rxMsg = tv
			stat.Reply = []byte(testMsg)
			stat.Status = api.DeliveryCompleted
			return stat
		})
	defer svc.Stop()

	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	sm := sessions.GetSessionManager()
	cs, err := sm.NewSession(agentID, "remote addr", "")
	assert.NoError(t, err)
	assert.NotNil(t, cs)

	// 2b. connect a client
	//sessionToken, err := dummyAuthenticator.Login(testLogin, testPassword, "")
	//require.NoError(t, err)
	//cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	//cl.ConnectWithToken(agentID, sessionToken)
	cl := httpclient.NewHttpSSEClient(hostPort, testLogin, certBundle.CaCert)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 3. publish two events
	// the path must match that in the config
	//vars := map[string]string{"thingID": thingID, "key": eventKey}
	//eventPath := utils.Substitute(vocab.PostEventPath, vars)
	//_, err = cl.Post(eventPath, testMsg)
	//_, err = cl.Post(eventPath, testMsg)
	stat := cl.PubEvent(thingID, eventKey, []byte(testMsg))
	require.Empty(t, stat.Error)
	stat = cl.PubEvent(thingID, eventKey, []byte(testMsg))
	require.Empty(t, stat.Error)

	// 4. verify that the handler received it
	assert.NoError(t, err)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, vocab.MessageTypeEvent, rxMsg.MessageType)
		assert.Equal(t, testMsg, string(rxMsg.Data))
	}

	// 5. publish an action
	//vars := map[string]string{"thingID": thingID, "key": actionKey}
	//actionPath := utils.Substitute(vocab.PostActionPath, vars)
	//_, err = cl.Post(actionPath, testMsg)
	stat, err = cl.PubAction(thingID, actionKey, []byte(testMsg))
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 100)
	assert.NotEmpty(t, stat.Reply)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)
		assert.Equal(t, testMsg, string(rxMsg.Data))
	}

	cl.Disconnect()
}

// Test publish subscribe using sse
func TestPubSubSSE(t *testing.T) {
	t.Log("TestPubSubSSE")
	var rxMsg *things.ThingMessage
	var testMsg = "hello world"
	//var agentID = "agent1"
	var thingID = "thing1"
	//var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the transport
	var svc *httpstransport.HttpsTransport
	svc = startHttpsBinding(
		func(tv *things.ThingMessage) (stat api.DeliveryStatus) {
			// broadcast event to subscribers
			slog.Info("broadcasting event")
			stat = svc.SendEvent(tv)
			return stat
		})
	defer svc.Stop()

	// 2. connect with a client
	cl := httpclient.NewHttpSSEClient(hostPort, testLogin, certBundle.CaCert)
	token, err := cl.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	defer cl.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// 3. register the handler for events
	cl.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		rxMsg = msg
		stat.Status = api.DeliveryCompleted
		return stat
	})

	// 4. publish an event using the hub client, the server will invoke the message handler
	// which in turn will publish this to the listeners over sse, including this client.
	stat := cl.PubEvent(thingID, eventKey, []byte(testMsg))
	assert.Empty(t, stat.Error)
	assert.Equal(t, api.DeliveryCompleted, stat.Status)
	time.Sleep(time.Millisecond * 10)
	//
	require.NotNil(t, rxMsg)
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, testLogin, rxMsg.SenderID)
	assert.Equal(t, eventKey, rxMsg.Key)

}