package httpsbinding_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/authn/jwtauth"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

const testPort = 10023

var certBundle = certs.CreateTestCertBundle()
var hostPort = fmt.Sprintf("localhost:%d", testPort)

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
		func(tv *thing.ThingValue) ([]byte, error) {
			return nil, nil
		})
	err := svc.Start()
	assert.NoError(t, err)
	svc.Stop()
}

// Test publishing an event
func TestPubEvent(t *testing.T) {
	var rxMsg *thing.ThingValue
	var testMsg = "hello world"
	var testReply = []byte("reply1")
	var agentID = "agent1"
	var thingID = "thing1"
	var eventKey = "key1"

	// 1. start the binding
	config := httpsbinding.NewHttpsBindingConfig()
	config.Port = testPort
	svc := httpsbinding.NewHttpsBinding(&config,
		certBundle.ServerKey, certBundle.ServerCert, certBundle.CaCert,
		func(tv *thing.ThingValue) ([]byte, error) {
			rxMsg = tv
			return testReply, nil
		})
	err := svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// 2a. create a session for connecting a client
	// (normally this happens when a session token is issued on authentication)
	sm := httpsbinding.GetSessionManager()
	cs, err := sm.NewSession(agentID, "remote addr")
	assert.NoError(t, err)
	assert.NotNil(t, cs)

	// 2b. connect a client
	sessionToken, err := jwtauth.CreateSessionToken(agentID, cs.GetSessionID(), certBundle.ServerKey, 10)
	require.NoError(t, err)
	cl := tlsclient.NewTLSClient(hostPort, certBundle.CaCert)
	cl.ConnectWithJwtAccessToken(agentID, sessionToken)

	// 3. publish an event
	eventPath := fmt.Sprintf("/event/%s/%s/%s", agentID, thingID, eventKey)
	resp, err := cl.Post(eventPath, testMsg)

	// 4. verify that the handler received it
	assert.NoError(t, err)
	assert.Equal(t, testReply, resp)
	if assert.NotNil(t, rxMsg) {
		assert.Equal(t, testMsg, string(rxMsg.Data))
	}

	cl.Close()
}
