package natsserver

import (
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"testing"
	"time"
)

func TestStartStopNKeys(t *testing.T) {
	var rxMsg string
	certBundle := certs.CreateTestCertBundle()
	homeDir := path.Join(os.TempDir(), "nats-server-test")
	logging.SetLogging("info", "")
	f := svcconfig.GetFolders(homeDir, false)

	s := NewNatsNKeyServer()
	cfg := &NatsServerConfig{
		Port: 9990,
		//ServerCert: certBundle.ServerCert, //auto generate
		CaCert: certBundle.CaCert,
		CaKey:  certBundle.CaKey,
		Debug:  true,
	}
	err := cfg.Setup(f.Certs, f.Stores, false)
	require.NoError(t, err)
	clientURL, err := s.Start(cfg)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// use the built-in service key
	c, err := s.ConnectInProc("testnkeysservice", nil)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	_, err = c.Subscribe("things.>", func(m *nats.Msg) {
		rxMsg = string(m.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	require.NoError(t, err)
	err = c.Publish("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

	c.Close()

	// add a service with its own key
	kp2, _ := nkeys.CreateUser()
	kp2Pub, _ := kp2.PublicKey()
	err = s.AddService("service2", kp2Pub)
	require.NoError(t, err)
	c, err = s.ConnectInProc("service2", kp2)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	_, err = c.Subscribe("things.service2.>", func(m *nats.Msg) {})
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	c.Close()
}
