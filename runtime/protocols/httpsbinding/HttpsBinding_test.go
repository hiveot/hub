package httpsbinding_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func startHttpsBinding(msgHandler func(message *thing.ThingMessage) ([]byte, error)) *httpsbinding.HttpsBinding {
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
	err := svc.Start(func(tv *thing.ThingMessage) ([]byte, error) {
		return nil, nil
	})
	assert.NoError(t, err)
	svc.Stop()
}

// Test publishing an event
func TestPubEvent(t *testing.T) {
	var rxMsg *thing.ThingMessage
	var testMsg = "hello world"
	var agentID = "agent1"
	var thingID = "thing1"
	var eventKey = "key1"

	// 1. start the binding
	svc := startHttpsBinding(
		func(tv *thing.ThingMessage) ([]byte, error) {
			assert.Equal(t, vocab.MessageTypeEvent, tv.MessageType)
			rxMsg = tv
			return nil, nil
		})
	defer svc.Stop()

	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	sm := sessions.GetSessionManager()
	cs, err := sm.NewSession(agentID, "remote addr", "")
	assert.NoError(t, err)
	assert.NotNil(t, cs)

	// 2b. connect a client
	sessionToken, err := dummyAuthenticator.Login(testLogin, testPassword, "")
	//sessionToken, err := jwtauth.CreateSessionToken(agentID, cs.GetSessionID(), certBundle.ServerKey, 10)
	require.NoError(t, err)

	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert, time.Second*120)
	cl.ConnectWithToken(agentID, sessionToken)

	// 3. publish two events
	// the path must match that in the config
	vars := map[string]string{"thingID": thingID, "key": eventKey}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)
	_, err = cl.Post(eventPath, testMsg)
	_, err = cl.Post(eventPath, testMsg)

	// 4. verify that the handler received it
	assert.NoError(t, err)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, testMsg, string(rxMsg.Data))
	}

	cl.Close()
	time.Sleep(time.Millisecond * 100)
}

func TestLoginRefresh(t *testing.T) {
	svc := startHttpsBinding(
		func(tv *thing.ThingMessage) ([]byte, error) {
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
		func(tv *thing.ThingMessage) ([]byte, error) {
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

// TODO: testcase for every single endpoint...

func TestBadRefresh(t *testing.T) {
	svc := startHttpsBinding(
		func(tv *thing.ThingMessage) ([]byte, error) {
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
