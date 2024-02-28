package mqttmsgserver_test

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
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
var TestDevice1Keys = keys.NewKey(keys.KeyTypeECDSA)
var TestDevice1PrivPEM = TestDevice1Keys.ExportPrivate()
var TestDevice1PubPEM = TestDevice1Keys.ExportPublic()

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)
var TestUser2ID = "user2"
var TestUser2Keys = keys.NewKey(keys.KeyTypeECDSA)
var TestUser2Key = TestUser2Keys.PrivateKey()
var TestUser2PubPEM = TestUser2Keys.ExportPublic()
var TestUser2Pass = "pass2"
var TestUser2bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser2Pass), 0)

var TestAdminUserID = "admin"

var TestAdminUserKey = keys.NewKey(keys.KeyTypeECDSA)
var TestAdminUserPubPEM = TestAdminUserKey.ExportPublic()
var TestAdminUserPrivPEM = TestAdminUserKey.ExportPrivate()

var TestService1ID = "service1"
var TestService1Key = keys.NewKey(keys.KeyTypeECDSA)
var TestService1PubPEM = TestService1Key.ExportPublic()

var adminAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestAdminUserID,
	ClientType: authapi.ClientTypeUser,
	PubKey:     TestAdminUserPubPEM,
	Role:       authapi.ClientRoleAdmin,
}
var deviceAuthInfo = msgserver.ClientAuthInfo{
	ClientID:   TestDevice1ID,
	ClientType: authapi.ClientTypeDevice,
	PubKey:     TestDevice1PubPEM,
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
		PubKey:       TestUser2PubPEM,
		PasswordHash: string(TestUser2bcrypt),
		Role:         authapi.ClientRoleOperator,
	},
	{
		ClientID:   TestService1ID,
		ClientType: authapi.ClientTypeService,
		PubKey:     TestService1PubPEM,
		Role:       authapi.ClientRoleAdmin,
	},
}

func newTransport(srv msgserver.IMsgServer, clientID string, caCert *x509.Certificate) transports.IHubTransport {
	url, _, _ := srv.GetServerURLs()
	//cl := mqtttransport_org.NewMqttTransportOrg(url, clientID, caCert)
	cl := mqtttransport.NewMqttTransport(url, clientID, caCert)
	return cl
}

func startServer(withAuth bool) (msgserver.IMsgServer, *certs.TestCertBundle) {
	srv, certBundle, err := testenv.StartMqttTestServer()
	if err != nil {
		panic("failed to start mqtt server")
	}
	if withAuth {
		err = srv.ApplyAuth(mqttTestClients)
		if err != nil {
			panic("failed to apply auth")
		}
	}
	return srv, &certBundle
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
	cl := newTransport(srv, TestUser1ID, certBundle.CaCert)
	err = cl.ConnectWithPassword(TestUser1Pass)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithToken(t *testing.T) {

	// setup
	srv, certBundle := startServer(true)
	defer srv.Stop()

	// admin is in the test clients with a public key
	adminToken, err := srv.CreateToken(adminAuthInfo)
	require.NoError(t, err)
	err = srv.ValidateToken(TestAdminUserID, adminToken, "", "")
	require.NoError(t, err)

	// login with token should succeed
	tp1 := newTransport(srv, TestAdminUserID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestAdminUserKey, adminToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	tp1.Disconnect()
}
func TestConnectBadCredentials(t *testing.T) {
	slog.Info("--- TestConnectBadCredentials start")
	defer slog.Info("--- TestConnectBadCredentials end")

	srv, certBundle, err := testenv.StartMqttTestServer()
	require.NoError(t, err)
	defer srv.Stop()
	err = srv.ApplyAuth(mqttTestClients)

	//key, _ := certs.CreateECDSAKeys()
	cl := newTransport(srv, TestUser1ID, certBundle.CaCert)
	cl.SetConnectHandler(func(stat transports.HubTransportStatus) {
		go cl.Disconnect()
	})
	err = cl.ConnectWithPassword("wrong password")
	assert.Error(t, err)
}
func TestMqttServerPubSub(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"

	srv, certBundle := startServer(true)
	defer srv.Stop()

	// create a key pair
	kp := srv.CreateKeyPair()
	assert.NotEmpty(t, kp)

	// connect and perform a pub/sub
	adminToken, err := srv.CreateToken(adminAuthInfo)
	tp1 := newTransport(srv, TestAdminUserID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestAdminUserKey, adminToken)
	require.NoError(t, err)
	defer tp1.Disconnect()
	tp1.SetEventHandler(func(addr string, payload []byte) {
		slog.Info("received msg", "addr", addr, "payload", string(payload))
		rxChan <- string(payload)
	})
	tp1.SetRequestHandler(func(addr string, payload []byte) ([]byte, error, bool) {
		return nil, fmt.Errorf("not implemented"), false
	})

	subTopic := mqtttransport.MakeTopic(transports.MessageTypeEvent, TestDevice1ID, "t1", "test", "")
	err = tp1.Subscribe(subTopic)
	require.NoError(t, err)

	// a device publishes an event
	tp2 := newTransport(srv, TestDevice1ID, certBundle.CaCert)
	token, _ := srv.CreateToken(deviceAuthInfo)
	err = tp2.ConnectWithToken(TestDevice1Keys, token)
	pubTopic := mqtttransport.MakeTopic(transports.MessageTypeEvent, TestDevice1ID, "t1", "test", TestDevice1ID)
	err = tp2.PubEvent(pubTopic, []byte(msg))
	require.NoError(t, err)
	rxMsg := <-rxChan

	assert.Equal(t, msg, rxMsg)
}

func TestMqttServerRequest(t *testing.T) {
	rxChan := make(chan string, 1)
	msg := "hello world"

	// setup
	srv, certBundle := startServer(true)
	defer srv.Stop()

	// create a key pair
	kp := srv.CreateKeyPair()
	assert.NotEmpty(t, kp)

	// connect and perform a pub/sub
	deviceToken, err := srv.CreateToken(deviceAuthInfo)
	tp1 := newTransport(srv, TestDevice1ID, certBundle.CaCert)
	err = tp1.ConnectWithToken(TestDevice1Keys, deviceToken)
	require.NoError(t, err)
	defer tp1.Disconnect()

	tp1.SetEventHandler(func(addr string, payload []byte) {
		slog.Info("received msg", "addr", addr, "payload", string(payload))
		rxChan <- string(payload)
	})
	tp1.SetRequestHandler(func(addr string, payload []byte) ([]byte, error, bool) {
		slog.Info("received action", "addr", addr)
		rxChan <- string(payload)
		return payload, nil, false
	})

	topic := mqtttransport.MakeTopic(transports.MessageTypeAction, TestDevice1ID, "", "", "")
	err = tp1.Subscribe(topic)
	require.NoError(t, err)

	// publish a request and match the response
	tp2 := newTransport(srv, TestUser2ID, certBundle.CaCert)
	_ = TestUser2Key
	err = tp2.ConnectWithPassword(TestUser2Pass)
	require.NoError(t, err)
	defer tp2.Disconnect()

	addr2 := mqtttransport.MakeTopic(transports.MessageTypeAction,
		TestDevice1ID, "thing1", "action1", TestUser2ID)
	reply, err := tp2.PubRequest(addr2, []byte(msg))
	require.NoError(t, err)
	assert.Equal(t, msg, string(reply))

	rxMsg := <-rxChan
	assert.Equal(t, msg, rxMsg)
}
