package httpsbinding_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports/httptransport"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
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

func (d *DummyAuthenticator) Login(clientID string, password string, sessionID string) (token string, err error) {
	if password == testPassword && clientID == testLogin {
		return testToken, nil
	}
	return "", fmt.Errorf("Invalid login")
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
func startHttpsBinding(msgHandler func(message *things.ThingMessage) ([]byte, error)) *httpsbinding.HttpsBinding {
	config := httpsbinding.NewHttpsBindingConfig()
	config.Port = testPort
	svc := httpsbinding.NewHttpsBinding(&config,
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
	sessionToken, err := dummyAuthenticator.Login(testLogin, testPassword, "")

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
	config := httpsbinding.NewHttpsBindingConfig()
	config.Port = testPort
	svc := httpsbinding.NewHttpsBinding(&config,
		certBundle.ClientKey, certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator,
	)
	err := svc.Start(func(tv *things.ThingMessage) ([]byte, error) {
		return nil, nil
	})
	assert.NoError(t, err)
	svc.Stop()
}

func TestLoginRefresh(t *testing.T) {
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) ([]byte, error) {
			assert.Fail(t, "should not get here")
			return nil, nil
		})
	defer svc.Stop()

	// the dummy authenticator accepts only the testLogin and testPassword
	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	token, err := cl.ConnectWithPassword(testLogin, testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// refresh should succeed
	token, err = cl.RefreshToken("")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// end the session
	err = cl.Logout()
	assert.NoError(t, err)

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the authenticator doesn't care.
	cl.ConnectWithToken(testLogin, token)
	token2, err := cl.RefreshToken("")
	assert.NoError(t, err)
	assert.NotEmpty(t, token2)

	// end the session
	err = cl.Logout()
	assert.NoError(t, err)
}

func TestBadLogin(t *testing.T) {
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) ([]byte, error) {
			assert.Fail(t, "should not get here")
			return nil, nil
		})
	defer svc.Stop()

	// check if this test still works with a valid login
	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	token, err := cl.ConnectWithPassword(testLogin, testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// failed logins
	token, err = cl.ConnectWithPassword(testLogin, "badpass")
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl.ConnectWithPassword("badID", testPassword)
	assert.Error(t, err)
	assert.Empty(t, token)
	// missing input
	loginURL := fmt.Sprintf("https://%s%s", hostPort, vocab.PostLoginPath)
	resp, err := cl.Invoke("POST", loginURL, "")
	assert.Error(t, err)
	assert.Empty(t, resp)
}

func TestBadRefresh(t *testing.T) {
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) ([]byte, error) {
			assert.Fail(t, "should not get here")
			return nil, nil
		})
	defer svc.Stop()

	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)

	// set the token
	cl.ConnectWithToken(testLogin, "badtoken")
	token, err := cl.RefreshToken("")
	assert.Error(t, err)
	assert.Empty(t, token)

	// get a valid token and refresh with a bad clientid
	token, err = cl.ConnectWithPassword(testLogin, testPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	cl.ConnectWithToken("badlogin", token)
	token, err = cl.RefreshToken("")
	assert.Error(t, err)
	assert.Empty(t, token)
}

// Test posting an event
func TestPostEventAction(t *testing.T) {
	var rxMsg *things.ThingMessage
	var testMsg = "hello world"
	var agentID = "agent1"
	var thingID = "thing1"
	var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	svc := startHttpsBinding(
		func(tv *things.ThingMessage) ([]byte, error) {
			rxMsg = tv
			return []byte(testMsg), nil
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
	tp := httptransport.NewHttpTransport(
		hostPort, vocab.ConnectSSEPath, testLogin, certBundle.CaCert)
	cl := hubclient.NewHubClientFromTransport(tp, testLogin)
	err = cl.ConnectWithPassword(testPassword)

	// 3. publish two events
	// the path must match that in the config
	//vars := map[string]string{"thingID": thingID, "key": eventKey}
	//eventPath := utils.Substitute(vocab.PostEventPath, vars)
	//_, err = cl.Post(eventPath, testMsg)
	//_, err = cl.Post(eventPath, testMsg)
	err = cl.PubEvent(thingID, eventKey, []byte(testMsg))
	err = cl.PubEvent(thingID, eventKey, []byte(testMsg))

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
	reply, err := cl.PubAction(agentID, thingID, actionKey, []byte(testMsg))
	time.Sleep(time.Millisecond * 100)
	assert.NoError(t, err)
	assert.NotEmpty(t, reply)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)
		assert.Equal(t, testMsg, string(rxMsg.Data))
	}

	//cl.Close()
	cl.Disconnect()
}

// Test publish subscribe using sse
func TestPubSubSSE(t *testing.T) {
	var rxMsg *things.ThingMessage
	var testMsg = "hello world"
	//var agentID = "agent1"
	var thingID = "thing1"
	//var actionKey = "action1"
	var eventKey = "event11"

	// 1. start the binding
	var svc *httpsbinding.HttpsBinding
	svc = startHttpsBinding(
		func(tv *things.ThingMessage) ([]byte, error) {
			// broadcast event to subscribers
			slog.Info("broadcasting event")
			svc.SendEvent(tv)
			return nil, nil
		})
	defer svc.Stop()

	// 2. connect with a client
	tp := httptransport.NewHttpTransport(
		hostPort, vocab.ConnectSSEPath, testLogin, certBundle.CaCert)
	hc := hubclient.NewHubClientFromTransport(tp, testLogin)
	err := hc.ConnectWithPassword(testPassword)
	require.NoError(t, err)
	defer hc.Disconnect()

	// give the client time to establish a sse connection
	time.Sleep(time.Millisecond * 3)

	// 3. register tha handler for events
	hc.SetEventHandler(func(msg *things.ThingMessage) {
		rxMsg = msg
	})

	// 4. publish an event using the hub client, the server will invoke the message handler
	// which in turn will publish this to the listeners over sse, including this client.
	err = hc.PubEvent(thingID, eventKey, []byte(testMsg))
	assert.NoError(t, err)
	time.Sleep(time.Second * 3)
	//
	require.NotNil(t, rxMsg)
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, testLogin, rxMsg.SenderID)
	assert.Equal(t, eventKey, rxMsg.Key)
}
