package mqttmsgserver_test

import (
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/hubclient/mqtthubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// ID
var TestDevice1ID = "device1"
var TestDevice1Key, TestDevice1Pub = certs.CreateECDSAKeys()

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)
var TestUser2ID = "user2"
var TestUser2Key, TestUser2Pub = certs.CreateECDSAKeys()
var TestUser2Pass = "pass2"
var TestUser2bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser2Pass), 0)

var TestAdminUserID = "admin"

var TestAdminUserKey, TestAdminUserPub = certs.CreateECDSAKeys()

var TestService1ID = "service1"
var TestService1Key, TestService1Pub = certs.CreateECDSAKeys()

var adminAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestAdminUserID,
	ClientType: auth2.ClientTypeUser,
	PubKey:     TestAdminUserPub,
	Role:       auth2.ClientRoleAdmin,
}
var deviceAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestDevice1ID,
	ClientType: auth2.ClientTypeDevice,
	PubKey:     TestDevice1Pub,
	Role:       auth2.ClientRoleDevice,
}
var mqttTestClients = []msgserver.ClientAuthInfo{
	adminAuthInfo,
	deviceAuthInfo,
	{
		ClientID:     TestUser1ID,
		ClientType:   auth2.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         auth2.ClientRoleViewer,
	},
	{
		ClientID:     TestUser2ID,
		ClientType:   auth2.ClientTypeUser,
		PubKey:       TestUser2Pub,
		PasswordHash: string(TestUser2bcrypt),
		Role:         auth2.ClientRoleOperator,
	},
	{
		ClientID:   TestService1ID,
		ClientType: auth2.ClientTypeService,
		PubKey:     TestService1Pub,
		Role:       auth2.ClientRoleAdmin,
	},
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestConnectWithCert(t *testing.T) {
	slog.Info("--- TestConnectWithCert start")
	defer slog.Info("--- TestConnectWithCert end")

	srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	_ = srv.ApplyAuth(mqttTestClients)

	key, _ := certs.CreateECDSAKeys()
	clientCert, err := certs.CreateClientCert(TestUser1ID, auth2.ClientRoleAdmin,
		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
	require.NoError(t, err)
	serverURL, _, _ := srv.GetServerURLs()
	cl := mqtthubclient.NewMqttHubClient(serverURL, TestUser1ID, key, certBundle.CaCert)
	clientTLS := certs.X509CertToTLS(clientCert, key)
	err = cl.ConnectWithCert(*clientTLS)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	key, _ := certs.CreateECDSAKeys()
	serverURL, _, _ := srv.GetServerURLs()
	cl := hubconnect.NewHubClient(serverURL, TestUser1ID, key, certBundle.CaCert, "mqtt")
	err = cl.ConnectWithPassword(TestUser1Pass)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithToken(t *testing.T) {

	// setup
	srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	// admin is in the test clients with a public key
	adminToken, err := srv.CreateToken(adminAuthInfo)
	require.NoError(t, err)
	err = srv.ValidateToken(TestAdminUserID, adminToken, "", "")
	require.NoError(t, err)

	// login with token should succeed
	serverURL, _, _ := srv.GetServerURLs()
	hc1 := hubconnect.NewHubClient(serverURL, TestAdminUserID, TestAdminUserKey, certBundle.CaCert, "mqtt")
	err = hc1.ConnectWithToken(adminToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	hc1.Disconnect()
}

func TestMqttServerPubSub(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"
	srv, certBundle, err := testenv.StartMqttTestServer()
	_ = certBundle
	require.NoError(t, err)

	//srv := service.NewMqttMsgServer(&cfg, auth.DefaultRolePermissions)
	//clientURL, err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()
	serverURL, _, _ := srv.GetServerURLs()
	assert.NotEmpty(t, serverURL)
	err = srv.ApplyAuth(mqttTestClients)
	require.NoError(t, err)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	adminToken, err := srv.CreateToken(adminAuthInfo)
	hc1 := hubconnect.NewHubClient(serverURL, TestAdminUserID, TestAdminUserKey, certBundle.CaCert, "mqtt")
	err = hc1.ConnectWithToken(adminToken)
	require.NoError(t, err)
	defer hc1.Disconnect()
	subTopic := mqtthubclient.MakeTopic(vocab.MessageTypeEvent, TestDevice1ID, "t1", "test", "")
	sub1, err := hc1.Sub(subTopic, func(addr string, data []byte) {
		slog.Info("received msg", "addr", addr, "data", string(data))
		rxChan <- string(data)
	})
	require.NoError(t, err)
	defer sub1.Unsubscribe()

	// a device publishes an event
	hc2 := hubconnect.NewHubClient(serverURL, TestDevice1ID, TestDevice1Key, certBundle.CaCert, "mqtt")
	token, _ := srv.CreateToken(deviceAuthInfo)
	err = hc2.ConnectWithToken(token)
	pubTopic := mqtthubclient.MakeTopic(vocab.MessageTypeEvent, TestDevice1ID, "t1", "test", TestDevice1ID)
	err = hc2.Pub(pubTopic, []byte(msg))
	require.NoError(t, err)
	rxMsg := <-rxChan

	assert.Equal(t, msg, rxMsg)
}

func TestMqttServerRequest(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"

	// setup the server with test clients
	srv, certBundle, err := testenv.StartMqttTestServer()
	_ = certBundle
	require.NoError(t, err)
	err = srv.ApplyAuth(mqttTestClients)
	require.NoError(t, err)
	defer srv.Stop()
	serverURL, _, _ := srv.GetServerURLs()
	assert.NotEmpty(t, serverURL)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	deviceToken, err := srv.CreateToken(deviceAuthInfo)
	hc := hubconnect.NewHubClient(serverURL, TestDevice1ID, TestDevice1Key, certBundle.CaCert, "mqtt")
	err = hc.ConnectWithToken(deviceToken)
	require.NoError(t, err)
	defer hc.Disconnect()

	sub2, err := hc.SubActions("thing1", func(ar *hubclient.RequestMessage) error {
		slog.Info("received action", "name", ar.Name)
		rxChan <- string(ar.Payload)
		err2 := ar.SendReply(ar.Payload, nil)
		assert.NoError(t, err2)
		return nil
	})
	defer sub2.Unsubscribe()

	// user publishes a device action
	hc2 := mqtthubclient.NewMqttHubClient(serverURL, TestUser2ID, TestUser2Key, certBundle.CaCert)
	err = hc2.ConnectWithPassword(TestUser2Pass)
	require.NoError(t, err)
	defer hc2.Disconnect()
	reply, err := hc2.PubAction("device1", "thing1", "action1", []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply.Payload))

	rxMsg := <-rxChan
	assert.Equal(t, msg, rxMsg)
}
