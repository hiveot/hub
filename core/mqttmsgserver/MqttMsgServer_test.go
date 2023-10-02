package mqttmsgserver_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/mqttmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nkeys"
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

// var TestDevice1NKey, _ = nkeys.CreateUser()
// var TestDevice1NPub, _ = TestDevice1NKey.PublicKey()
var TestDevice1Key, TestDevice1Pub = certs.CreateECDSAKeys()

//var TestThing1ID = "thing1"
//var TestThing2ID = "thing2"

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)

var TestAdminUserID = "admin"

// var TestAdminUserNKey, _ = nkeys.CreateUser()
// var TestAdminUserNPub, _ = TestAdminUserNKey.PublicKey()
var TestAdminUserKey, TestAdminUserPub = certs.CreateECDSAKeys()

var TestService1ID = "service1"
var TestService1NKey, _ = nkeys.CreateUser()
var TestService1NPub, _ = TestService1NKey.PublicKey()

//var TestService1Key, TestService1Pub = certs.CreateECDSAKeys()

var mqttTestClients = []msgserver.ClientAuthInfo{
	{
		ClientID:   TestAdminUserID,
		ClientType: auth.ClientTypeUser,
		PubKey:     TestAdminUserPub,
		Role:       auth.ClientRoleAdmin,
	},
	{
		ClientID:   TestDevice1ID,
		ClientType: auth.ClientTypeDevice,
		PubKey:     TestDevice1Pub,
		Role:       auth.ClientRoleDevice,
	},
	{
		ClientID:     TestUser1ID,
		ClientType:   auth.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         auth.ClientRoleViewer,
	},
	{
		ClientID:   TestService1ID,
		ClientType: auth.ClientTypeService,
		PubKey:     TestService1NPub,
		Role:       auth.ClientRoleAdmin,
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

	serverURL, srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	_ = srv.ApplyAuth(mqttTestClients)

	key, _ := certs.CreateECDSAKeys()
	clientCert, err := certs.CreateClientCert(TestUser1ID, auth.ClientRoleAdmin,
		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
	require.NoError(t, err)
	cl := mqtthubclient.NewMqttHubClient(serverURL, TestUser1ID, key, certBundle.CaCert)
	clientTLS := certs.X509CertToTLS(clientCert, key)
	err = cl.ConnectWithCert(*clientTLS)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	serverURL, srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	key, _ := certs.CreateECDSAKeys()
	cl := hubcl.NewHubClient(serverURL, TestUser1ID, key, certBundle.CaCert, "mqtt")
	err = cl.ConnectWithPassword(TestUser1Pass)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithToken(t *testing.T) {

	// setup
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer()
	msrv := srv.(*service.MqttMsgServer)
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	// admin is in the test clients with a public key
	adminInfo, err := msrv.GetClientAuth(TestAdminUserID)
	require.NoError(t, err)
	adminToken, err := msrv.CreateToken(adminInfo)
	require.NoError(t, err)
	err = msrv.ValidateToken(TestAdminUserID, adminToken, "", "")
	require.NoError(t, err)

	// login with token should succeed
	hc1 := hubcl.NewHubClient(serverURL, TestAdminUserID, TestAdminUserKey, certBundle.CaCert, "mqtt")
	err = hc1.ConnectWithToken(adminToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	hc1.Disconnect()
}

func TestMqttServerPubSub(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer()
	_ = certBundle
	require.NoError(t, err)

	//srv := service.NewMqttMsgServer(&cfg, auth.DefaultRolePermissions)
	//clientURL, err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, serverURL)
	err = srv.ApplyAuth(mqttTestClients)
	require.NoError(t, err)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc(TestAdminUserID)
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
	serverURL, srv, certBundle, err := testenv.StartMqttTestServer()
	_ = certBundle
	require.NoError(t, err)
	err = srv.ApplyAuth(mqttTestClients)
	require.NoError(t, err)
	defer srv.Stop()
	assert.NotEmpty(t, serverURL)

	// create a key pair
	kp, pubKey := srv.CreateKP()
	assert.NotEmpty(t, kp)
	assert.NotEmpty(t, pubKey)

	// connect and perform a pub/sub
	hc, err := srv.ConnectInProc(TestDevice1ID)
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
