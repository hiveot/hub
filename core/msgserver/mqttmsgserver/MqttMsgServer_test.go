package mqttmsgserver_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
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
var TestDevice1KP, _ = certs.PrivateKeyToPEM(TestDevice1Key)

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)
var TestUser2ID = "user2"
var TestUser2Key, TestUser2Pub = certs.CreateECDSAKeys()
var TestUser2Pass = "pass2"
var TestUser2bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser2Pass), 0)

var TestAdminUserID = "admin"

var TestAdminUserKey, TestAdminUserPub = certs.CreateECDSAKeys()
var TestAdminUserKP, _ = certs.PrivateKeyToPEM(TestAdminUserKey)

var TestService1ID = "service1"
var TestService1Key, TestService1Pub = certs.CreateECDSAKeys()

var adminAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestAdminUserID,
	ClientType: authapi.ClientTypeUser,
	PubKey:     TestAdminUserPub,
	Role:       authapi.ClientRoleAdmin,
}
var deviceAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestDevice1ID,
	ClientType: authapi.ClientTypeDevice,
	PubKey:     TestDevice1Pub,
	Role:       authapi.ClientRoleDevice,
}
var mqttTestClients = []msgserver.ClientAuthInfo{
	adminAuthInfo,
	deviceAuthInfo,
	{
		ClientID:     TestUser1ID,
		ClientType:   authapi.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         authapi.ClientRoleViewer,
	},
	{
		ClientID:     TestUser2ID,
		ClientType:   authapi.ClientTypeUser,
		PubKey:       TestUser2Pub,
		PasswordHash: string(TestUser2bcrypt),
		Role:         authapi.ClientRoleOperator,
	},
	{
		ClientID:   TestService1ID,
		ClientType: authapi.ClientTypeService,
		PubKey:     TestService1Pub,
		Role:       authapi.ClientRoleAdmin,
	},
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

//func TestConnectWithCert(t *testing.T) {
//	slog.Info("--- TestConnectWithCert start")
//	defer slog.Info("--- TestConnectWithCert end")
//
//	srv, certBundle, err := testenv.StartMqttTestServer()
//	require.NoError(t, err)
//	defer srv.Stop()
//	_ = srv.ApplyAuth(mqttTestClients)
//
//	key, _ := certs.CreateECDSAKeys()
//	clientCert, err := certs.CreateClientCert(TestUser1ID, authapi.ClientRoleAdmin,
//		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
//	require.NoError(t, err)
//	serverURL, _, _ := srv.GetServerURLs()
//	cl := mqtttransport.NewMqttTransport(serverURL, TestUser1ID, certBundle.CaCert)
//	clientTLS := certs.X509CertToTLS(clientCert, key)
//	err = cl.ConnectWithCert(*clientTLS)
//	assert.NoError(t, err)
//	defer cl.Disconnect()
//}

func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	//key, _ := certs.CreateECDSAKeys()
	serverURL, _, _ := srv.GetServerURLs()
	cl := mqtttransport.NewMqttTransport(serverURL, TestUser1ID, certBundle.CaCert)
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
	tp1 := mqtttransport.NewMqttTransport(serverURL, TestAdminUserID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestAdminUserKP, adminToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	tp1.Disconnect()
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
	serializedKP, pubKey := srv.CreateKeyPair()
	assert.NotEmpty(t, serializedKP)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	adminToken, err := srv.CreateToken(adminAuthInfo)
	tp1 := mqtttransport.NewMqttTransport(serverURL, TestAdminUserID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestAdminUserKP, adminToken)
	require.NoError(t, err)
	defer tp1.Disconnect()
	subTopic := mqtttransport.MakeTopic(vocab.MessageTypeEvent, TestDevice1ID, "t1", "test", "")
	sub1, err := tp1.Sub(subTopic, func(addr string, data []byte) {
		slog.Info("received msg", "addr", addr, "data", string(data))
		rxChan <- string(data)
	})
	require.NoError(t, err)
	defer sub1.Unsubscribe()

	// a device publishes an event
	tp2 := mqtttransport.NewMqttTransport(serverURL, TestDevice1ID, certBundle.CaCert)
	token, _ := srv.CreateToken(deviceAuthInfo)
	err = tp2.ConnectWithToken(TestDevice1KP, token)
	pubTopic := mqtttransport.MakeTopic(vocab.MessageTypeEvent, TestDevice1ID, "t1", "test", TestDevice1ID)
	err = tp2.Pub(pubTopic, []byte(msg))
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
	serializedKP, pubKey := srv.CreateKeyPair()
	assert.NotEmpty(t, serializedKP)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	deviceToken, err := srv.CreateToken(deviceAuthInfo)
	tp1 := mqtttransport.NewMqttTransport(serverURL, TestDevice1ID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestDevice1KP, deviceToken)
	require.NoError(t, err)
	defer tp1.Disconnect()

	topic := mqtttransport.MakeTopic(vocab.MessageTypeAction, TestDevice1ID, "", "", "")
	sub2, err := tp1.SubRequest(topic, func(addr string, payload []byte) (resp []byte, err error) {
		slog.Info("received action", "addr", addr)
		rxChan <- string(payload)
		return payload, nil
	})
	defer sub2.Unsubscribe()

	// user publishes a device action
	hc2 := mqtttransport.NewMqttTransport(serverURL, TestUser2ID, certBundle.CaCert)
	_ = TestUser2Key
	err = hc2.ConnectWithPassword(TestUser2Pass)
	require.NoError(t, err)
	defer hc2.Disconnect()
	addr2 := mqtttransport.MakeTopic(vocab.MessageTypeAction,
		TestDevice1ID, "thing1", "action1", TestUser2ID)
	reply, err := hc2.PubRequest(addr2, []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply))

	rxMsg := <-rxChan
	assert.Equal(t, msg, rxMsg)
}
