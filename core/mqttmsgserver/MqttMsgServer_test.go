package mqttmsgserver_test

import (
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/lib/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"testing"
	"time"
)

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestStartStopMqttServer(t *testing.T) {
	var rxMsg string
	msg := "hello world"
	cfg := mqttmsgserver.MqttServerConfig{}
	err := cfg.Setup("", "", false)
	require.NoError(t, err)
	srv := mqttmsgserver.NewMqttMsgServer(&cfg, nil)
	clientURL, err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, clientURL)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc("test")
	require.NoError(t, err)
	defer hc.Disconnect()

	sub1, err := hc.Sub("test", func(addr string, data []byte) {
		slog.Info("received msg", "addr", addr, "data", string(data))
		rxMsg = string(data)
	})
	require.NoError(t, err)
	defer sub1.Unsubscribe()

	err = hc.Pub("test", []byte(msg))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)

	assert.Equal(t, msg, rxMsg)
}
