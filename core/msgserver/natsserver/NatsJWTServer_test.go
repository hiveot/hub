package natsserver

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func TestStartStopJWT(t *testing.T) {
	homeDir := path.Join(os.TempDir(), "nats-server-test")
	f := svcconfig.GetFolders(homeDir, false)

	certBundle := certs.CreateTestCertBundle()
	s := NewNatsJWTServer()
	cfg := NatsServerConfig{
		Port:       9990,
		ServerCert: certBundle.ServerCert,
		CaCert:     certBundle.CaCert,
		CaKey:      certBundle.CaKey,
	}
	err := cfg.Setup(f.Certs, f.Stores, false)
	require.NoError(t, err)
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	c, err := s.ConnectInProc("testjwtservice")
	require.NoError(t, err)
	require.NotEmpty(t, c)
	c.Close()
}
