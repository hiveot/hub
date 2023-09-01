package natsnkeyserver_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartStopCallout(t *testing.T) {
	var coCount atomic.Int32
	// defined in NatsNKeyServer_test.go
	clientURL, s, _, _, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// this callout handler only accepts 'validuser' request
	chook, err := natsnkeyserver.EnableNatsCalloutHook(s, func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != "validuser" {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	_ = chook
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
	var newUser = "newUser"
	var newKey, _ = nkeys.CreateUser()

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(testenv.TestClients)
	assert.NoError(t, err)

	// this callout handler only accepts 'knownUser' request
	chook, err := natsnkeyserver.EnableNatsCalloutHook(s, func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		newKeyPub, _ := newKey.PublicKey()
		coCount.Add(1)
		claims, err := jwt.DecodeUserClaims(request.ConnectOptions.Token)
		if err != nil {
			return err
		} else if request.ClientInformation.Name != newUser {
			return fmt.Errorf("unknown user: %s", request.Name)
		} else if newKeyPub != claims.Subject {
			return fmt.Errorf("pubkey mismatch")
		}
		_ = claims
		return nil
	})
	_ = chook
	assert.NoError(t, err)

	// a directly added service should not invoke the callout handler
	// (added by nkey server test)
	c, err := s.ConnectInProc(testenv.TestService1ID)
	require.NoError(t, err)
	c.Disconnect()
	assert.Equal(t, int32(0), coCount.Load())

	newKeyPub, _ := newKey.PublicKey()
	loginToken, err := s.CreateToken(newUser, auth.ClientTypeUser, newKeyPub, time.Minute)
	require.NoError(t, err)
	// invoke callout by connecting with a new user
	c, err = natshubclient.ConnectWithJWT(clientURL, newKey, loginToken, certBundle.CaCert)
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
	err = s.ApplyAuth(testenv.TestClients)

	// this callout handler only accepts 'knownUser' request
	chook, err := natsnkeyserver.EnableNatsCalloutHook(s, func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != knownUser {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	_ = chook
	assert.NoError(t, err)

	// invoke callout by connecting with an invalid user
	newkey2, _ := nkeys.CreateUser()
	_, err = natshubclient.ConnectWithNKey(clientURL, "unknownuser", newkey2, certBundle.CaCert)
	require.Error(t, err)
	assert.Equal(t, int32(1), coCount.Load())
}

func TestCalloutToken(t *testing.T) {
	logrus.Infof("---TestToken start---")
	defer logrus.Infof("---TestToken end---")

	var coCount atomic.Int32
	var newUser = "newUser"
	var newKey, _ = nkeys.CreateUser()

	// setup
	clientURL, s, _, _, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(testenv.TestClients)
	assert.NoError(t, err)

	// this callout handler only accepts 'knownUser' request
	chook, err := natsnkeyserver.EnableNatsCalloutHook(s, func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		newKeyPub, _ := newKey.PublicKey()
		coCount.Add(1)
		claims, err := jwt.DecodeUserClaims(request.ConnectOptions.Token)
		if err != nil {
			return err
		} else if request.ClientInformation.Name != newUser {
			return fmt.Errorf("unknown user: %s", request.Name)
		} else if newKeyPub != claims.Subject {
			return fmt.Errorf("pubkey mismatch")
		}
		_ = claims
		return nil
	})
	_ = chook
	assert.NoError(t, err)

	token, err := s.CreateToken(testenv.TestUser2ID, auth.ClientTypeUser, testenv.TestUser2Pub, time.Minute)
	require.NoError(t, err)

	err = s.ValidateToken(testenv.TestUser2ID, testenv.TestUser2Pub, token, "", "")
	assert.NoError(t, err)
}
