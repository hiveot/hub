package natscoserver_test

import (
	"fmt"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"sync/atomic"
	"testing"
)

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}
func TestStartStopCallout(t *testing.T) {
	var coCount atomic.Int32
	// defined in NatsNKeyServer_test.go
	clientURL, s, _, _, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// this callout handler only accepts 'validuser' request
	err = s.EnableCalloutHandler(func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != "validuser" {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	assert.NoError(t, err)

	// core services do not use the callout handler
	c, err := s.ConnectInProc("testcalloutservice")
	require.NoError(t, err)
	require.NotEmpty(t, c)
	assert.Equal(t, int32(0), coCount.Load())
	c.Disconnect()
}

func TestValidCalloutAuthn(t *testing.T) {
	var coCount atomic.Int32
	var knownUser = "knownUser"

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuthn(testenv.TestClients)
	assert.NoError(t, err)
	err = s.ApplyAuthz(testenv.TestRoles)
	assert.NoError(t, err)

	// this callout handler only accepts 'knownUser' request
	err = s.EnableCalloutHandler(func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != knownUser {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	assert.NoError(t, err)

	// a directly added service should not invoke the callout handler
	// (added by nkey server test)
	c, err := s.ConnectInProc(testenv.TestService1ID)
	require.NoError(t, err)
	c.Disconnect()
	assert.Equal(t, int32(0), coCount.Load())

	// invoke callout by connecting with a new user
	newkey2, _ := nkeys.CreateUser()
	c, err = natshubclient.ConnectWithNKey(clientURL, knownUser, newkey2, certBundle.CaCert)
	require.NoError(t, err)

	c.Disconnect()
	assert.Equal(t, int32(1), coCount.Load())
}

func TestInValidCalloutAuthn(t *testing.T) {
	const knownUser = "knownuser"

	var coCount atomic.Int32

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)
	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuthn(testenv.TestClients)
	err = s.ApplyAuthz(testenv.TestRoles)

	// this callout handler only accepts 'knownUser' request
	err = s.EnableCalloutHandler(func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != knownUser {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	assert.NoError(t, err)

	// invoke callout by connecting with an invalid user
	newkey2, _ := nkeys.CreateUser()
	_, err = natshubclient.ConnectWithNKey(clientURL, "unknownuser", newkey2, certBundle.CaCert)
	require.Error(t, err)
	assert.Equal(t, int32(1), coCount.Load())
}
