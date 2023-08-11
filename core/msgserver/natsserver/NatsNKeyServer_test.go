package natsserver

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStartStopNKeys(t *testing.T) {
	certBundle := certs.CreateTestCertBundle()

	s := NewNatsNKeyServer(certBundle.ServerCert, certBundle.CaCert)
	cfg := NatsNKeysConfig{
		Port:       9990,
		ServerCert: certBundle.ServerCert,
		CaCert:     certBundle.CaCert,
	}
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// use the built-in service key
	c, err := s.ConnectInProc("testnkeysservice", nil)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	c.Close()

	// add a service with its own key
	kp2, _ := nkeys.CreateUser()
	kp2Pub, _ := kp2.PublicKey()
	err = s.AddService("service2", kp2Pub)
	require.NoError(t, err)
	c, err = s.ConnectInProc("service2", kp2)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	c.Close()
}
