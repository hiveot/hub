package mqttmsgserver_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
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
	rxChan := make(chan string, 1)
	msg := "hello world"
	cfg := mqttmsgserver.MqttServerConfig{}
	err := cfg.Setup("", "", false)
	require.NoError(t, err)

	srv := mqttmsgserver.NewMqttMsgServer(&cfg, auth.DefaultRolePermissions)
	clientURL, err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, clientURL)
	err = srv.ApplyAuth(testenv.MqttTestClients)
	require.NoError(t, err)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc(testenv.TestAdminUserID)
	require.NoError(t, err)
	defer hc.Disconnect()
	topic1 := "things/d1/t1/event/test"
	sub1, err := hc.Sub(topic1, func(addr string, data []byte) {
		slog.Info("received msg", "addr", addr, "data", string(data))
		rxChan <- string(data)
	})
	require.NoError(t, err)
	defer sub1.Unsubscribe()

	err = hc.Pub(topic1, []byte(msg))
	require.NoError(t, err)
	rxMsg := <-rxChan

	assert.Equal(t, msg, rxMsg)
}

func TestMqttServerRequest(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"

	// setup the server with test clients
	clientURL, srv, certBundle, err := testenv.StartTestServer("mqtt", true)
	_ = certBundle
	require.NoError(t, err)
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, clientURL)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc(testenv.TestDevice1ID)
	require.NoError(t, err)
	defer hc.Disconnect()

	sub2, err := hc.SubThingActions("thing1", func(ar *hubclient.ActionRequest) error {
		slog.Info("received action", "name", ar.ActionID)
		rxChan <- string(ar.Payload)
		err2 := ar.SendReply(ar.Payload, nil)
		assert.NoError(t, err2)
		return nil
	})
	defer sub2.Unsubscribe()

	reply, err := hc.PubThingAction("device1", "thing1", "action1", []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply.Payload))

	rxMsg := <-rxChan
	assert.Equal(t, msg, rxMsg)
}

func TestToken(t *testing.T) {

	// setup
	serverURL, srv, certBundle, err := testenv.StartTestServer("mqtt", true)
	msrv := srv.(*mqttmsgserver.MqttMsgServer)
	require.NoError(t, err)
	defer srv.Stop()

	// admin is in the test clients with a public key
	adminInfo, err := msrv.GetClientAuth(testenv.TestAdminUserID)
	require.NoError(t, err)
	adminToken, err := msrv.CreateToken(adminInfo)
	require.NoError(t, err)
	_, err = msrv.ValidateToken(testenv.TestAdminUserID, adminToken, "", "")
	require.NoError(t, err)

	// login with token should succeed
	hc1 := hubcl.NewHubClient(serverURL, testenv.TestAdminUserID, testenv.TestAdminUserKey, certBundle.CaCert, "mqtt")
	err = hc1.ConnectWithToken(adminToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	hc1.Disconnect()
}
