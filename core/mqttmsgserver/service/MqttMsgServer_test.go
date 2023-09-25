package service_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/mqttmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
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

func TestConnectWithCert(t *testing.T) {
	slog.Info("--- TestConnectWithCert start")
	defer slog.Info("--- TestConnectWithCert end")

	serverURL, srv, certBundle, err := testenv.StartMqttTestServer(true)
	require.NoError(t, err)
	defer srv.Stop()

	key, _ := certs.CreateECDSAKeys()
	clientCert, err := certs.CreateClientCert(testenv.TestUser1ID, auth.ClientRoleAdmin,
		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
	require.NoError(t, err)
	cl := mqtthubclient.NewMqttHubClient(serverURL, testenv.TestUser1ID, key, certBundle.CaCert)
	clientTLS := certs.X509CertToTLS(clientCert, key)
	err = cl.ConnectWithCert(*clientTLS)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	serverURL, srv, certBundle, err := testenv.StartMqttTestServer(true)
	require.NoError(t, err)
	defer srv.Stop()

	key, _ := certs.CreateECDSAKeys()
	cl := hubcl.NewHubClient(serverURL, testenv.TestUser1ID, key, certBundle.CaCert, "mqtt")
	err = cl.ConnectWithPassword(testenv.TestUser1Pass)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithToken(t *testing.T) {

	// setup
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer(true)
	msrv := srv.(*service.MqttMsgServer)
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

func TestMqttServerPubSub(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer(false)
	_ = certBundle
	require.NoError(t, err)

	//srv := service.NewMqttMsgServer(&cfg, auth.DefaultRolePermissions)
	//clientURL, err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, serverURL)
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
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer(true)
	_ = certBundle
	require.NoError(t, err)
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, serverURL)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc(testenv.TestDevice1ID)
	require.NoError(t, err)
	defer hc.Disconnect()

	sub2, err := hc.SubActions("thing1", func(ar *hubclient.RequestMessage) error {
		slog.Info("received action", "name", ar.ActionID)
		rxChan <- string(ar.Payload)
		err2 := ar.SendReply(ar.Payload, nil)
		assert.NoError(t, err2)
		return nil
	})
	defer sub2.Unsubscribe()

	reply, err := hc.PubAction("device1", "thing1", "action1", []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply.Payload))

	rxMsg := <-rxChan
	assert.Equal(t, msg, rxMsg)
}
