package natsserver

import (
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
)

func TestStartStopCallout(t *testing.T) {
	homeDir := path.Join(os.TempDir(), "nats-server-test")

	var coCount atomic.Int32
	certBundle := certs.CreateTestCertBundle()

	s := NewNatsCalloutServer()
	cfg := NatsServerConfig{
		Port:       9990,
		ServerCert: certBundle.ServerCert,
		CaCert:     certBundle.CaCert,
		CaKey:      certBundle.CaKey,
	}
	err := cfg.Setup(homeDir, false)
	require.NoError(t, err)
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// this callout handler only accepts 'validuser' request
	err = s.SetCalloutHandler(func(request *jwt.AuthorizationRequestClaims) error {
		slog.Info("received request")
		coCount.Add(1)
		if request.ClientInformation.Name != "validuser" {
			return fmt.Errorf("unknown user: %s", request.Name)
		}
		return nil
	})
	assert.NoError(t, err)

	// core services do not use the callout handler
	c, err := s.ConnectInProc("testcalloutservice", nil)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	assert.Equal(t, int32(0), coCount.Load())
	c.Close()

	// a new service should not invoke the callout handler
	newkey1, _ := nkeys.CreateUser()
	newkey1Pub, _ := newkey1.PublicKey()
	s.AddService("newservice", newkey1Pub)
	c, err = s.ConnectInProc("user1", newkey1)
	require.NoError(t, err)
	c.Close()
	assert.Equal(t, int32(0), coCount.Load())

	// invoke callout by connecting with a valid user
	newkey2, _ := nkeys.CreateUser()
	c, err = s.ConnectInProc("validuser", newkey2)
	require.NoError(t, err)
	c.Close()
	assert.Equal(t, int32(1), coCount.Load())

	// invoke callout by connecting with an invalid user
	// this will timeout as an unknown user has no inbox
	c, err = s.ConnectInProc("invaliduser", newkey2)
	require.Error(t, err)
	assert.Equal(t, int32(2), coCount.Load())
}
