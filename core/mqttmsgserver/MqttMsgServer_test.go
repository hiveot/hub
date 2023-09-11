package mqttmsgserver_test

import (
	"github.com/hiveot/hub/api/go/hubclient"
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

func TestMqttServerPubSub(t *testing.T) {
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

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

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

func TestMqttServerRequest(t *testing.T) {
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

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc("device1")
	require.NoError(t, err)
	defer hc.Disconnect()

	sub2, err := hc.SubThingActions("thing1", func(ar *hubclient.ActionRequest) error {
		slog.Info("received action", "name", ar.ActionID)
		rxMsg = string(ar.Payload)
		err2 := ar.SendReply(ar.Payload, nil)
		assert.NoError(t, err2)
		return nil
	})
	defer sub2.Unsubscribe()

	//topic2 := mqtthubclient.MakeThingActionTopic("device1", "thing1", "action1", "itsme")
	//reply := ""
	//err = hc.Pub(topic2, []byte(msg))
	reply, err := hc.PubThingAction("device1", "thing1", "action1", []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply))
	time.Sleep(time.Second)
	time.Sleep(time.Millisecond)

	assert.Equal(t, msg, rxMsg)
}
