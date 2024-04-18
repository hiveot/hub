package httpsbinding_test

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
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
	return testToken, nil
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

//---------

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
	config := httpsbinding.NewHttpsBindingConfig()
	config.Port = testPort
	svc := httpsbinding.NewHttpsBinding(&config,
		certBundle.ServerKey, certBundle.ServerCert, certBundle.CaCert,
		dummyAuthenticator)
	err := svc.Start(func(tv *thing.ThingMessage) ([]byte, error) {
		assert.Equal(t, vocab.MessageTypeEvent, tv.MessageType)
		rxMsg = tv
		return nil, nil
	})
	require.NoError(t, err)
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
	eventPath := utils.Substitute(vocab.AgentPostEventPath, vars)
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
