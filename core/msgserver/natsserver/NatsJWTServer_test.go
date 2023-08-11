package natsserver

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStartStopJWT(t *testing.T) {
	certBundle := certs.CreateTestCertBundle()
	s := NewNatsJWTServer(certBundle.ServerCert, certBundle.CaCert)
	cfg := NatsJWTConfig{
		Port:       9990,
		ServerCert: certBundle.ServerCert,
		CaCert:     certBundle.CaCert,
	}
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	c, err := s.ConnectInProc("testjwtservice")
	require.NoError(t, err)
	require.NotEmpty(t, c)
	c.Close()
}
