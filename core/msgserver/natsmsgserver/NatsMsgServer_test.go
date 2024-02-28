package natsmsgserver_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/natsmsgserver/service"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/natstransport"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"os"
	"testing"
	"time"
)

const withCallout = false

var TestDevice1ID = "device1"
var TestDevice1NKey, _ = nkeys.CreateUser()
var TestDevice1NPub, _ = TestDevice1NKey.PublicKey()

//var TestDevice1Key, TestDevice1Pub = certs.CreateECDSAKeys()

var TestThing1ID = "thing1"

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)

var TestAdminUserID = "admin"
var TestAdminUserNKey, _ = nkeys.CreateUser()
var TestAdminUserNPub, _ = TestAdminUserNKey.PublicKey()

//var TestAdminUserKey, TestAdminUserPub = certs.CreateECDSAKeys()

var TestService1ID = "service1"
var TestService1NKey, _ = nkeys.CreateUser()
var TestService1NPub, _ = TestService1NKey.PublicKey()

var NatsTestClients = []msgserver.ClientAuthInfo{
	{
		ClientID:   TestAdminUserID,
		ClientType: authapi.ClientTypeUser,
		PubKey:     TestAdminUserNPub,
		Role:       authapi.ClientRoleAdmin,
	},
	{
		ClientID:   TestDevice1ID,
		ClientType: authapi.ClientTypeDevice,
		PubKey:     TestDevice1NPub,
		Role:       authapi.ClientRoleDevice,
	},
	{
		ClientID:     TestUser1ID,
		ClientType:   authapi.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         authapi.ClientRoleViewer,
	},
	{
		ClientID:   TestService1ID,
		ClientType: authapi.ClientTypeService,
		PubKey:     TestService1NPub,
		Role:       authapi.ClientRoleService, // admin adds the $JS.API.INFO permissions
	},
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestStartStopNKeysServer(t *testing.T) {

	srv, certBundle, _, err := testenv.StartNatsTestServer(withCallout)

	require.NoError(t, err)
	defer srv.Stop()

	err = srv.ApplyAuth(NatsTestClients)
	require.NoError(t, err)

	// connect with test user
	//nc, err := srv.ConnectInProc("testnkeysservice", nil)
	serverURL, _, _ := srv.GetServerURLs()
	tp := natstransport.NewNatsTransport(serverURL, TestAdminUserID, certBundle.CaCert)
	err = tp.ConnectWithKey(TestAdminUserNKey)
	require.NoError(t, err)
	defer tp.Disconnect()

	// make sure jetstream is enabled for account
	js := tp.JS()
	ai, err := js.AccountInfo()
	require.NoError(t, err)
	require.NotEmpty(t, ai)
}

//func TestConnectWithCert(t *testing.T) {
//	slog.Info("--- TestConnectWithCert start")
//	defer slog.Info("--- TestConnectWithCert end")
//
//	// this only works with callout
//	srv, _, certBundle, err := testenv.StartNatsTestServer(true)
//	require.NoError(t, err)
//	defer srv.Stop()
//
//	// user1 used in this test must exist
//	_ = srv.ApplyAuth(NatsTestClients)
//
//	key, _ := certs.CreateECDSAKeys()
//	clientCert, err := certs.CreateClientCert(TestUser1ID, authapi.ClientRoleAdmin,
//		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
//	require.NoError(t, err)
//	serverURL, _, _ := srv.GetServerURLs()
//	cl := natshubtransport.NewNatsHubClient(serverURL, TestUser1ID, nil, certBundle.CaCert)
//	clientTLS := certs.X509CertToTLS(clientCert, key)
//	err = cl.ConnectWithCert(*clientTLS)
//	assert.NoError(t, err)
//	defer cl.Disconnect()
//}

func TestConnectWithNKey(t *testing.T) {

	slog.Info("--- TestConnectWithNKey start")
	defer slog.Info("--- TestConnectWithNKey end")
	rxChan := make(chan string, 1)

	srv, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer srv.Stop()

	// add several users, service and devices
	err = srv.ApplyAuth(NatsTestClients)
	require.NoError(t, err)

	// users subscribe to things
	serverURL, _, _ := srv.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(serverURL, TestService1ID, certBundle.CaCert)
	err = tp1.ConnectWithKey(TestService1NKey)
	require.NoError(t, err)
	defer tp1.Disconnect()

	tp1.SetEventHandler(func(addr string, payload []byte) {
		slog.Info("received msg", "addr", addr, "payload", string(payload))
		rxChan <- string(payload)
	})

	subSubj := natstransport.MakeSubject(
		transports.MessageTypeEvent, "", "", "", "")
	err = tp1.Subscribe(subSubj)
	assert.NoError(t, err)
	pubSubj := natstransport.MakeSubject(
		transports.MessageTypeEvent, TestService1ID, "thing1", "test", TestService1ID)
	err = tp1.PubEvent(pubSubj, []byte("hello world"))
	require.NoError(t, err)
	rxMsg := <-rxChan
	assert.Equal(t, "hello world", rxMsg)

}
func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	srv, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer srv.Stop()

	// add several users, service and devices
	err = srv.ApplyAuth(NatsTestClients)
	require.NoError(t, err)

	serverURL, _, _ := srv.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(serverURL, TestUser1ID, certBundle.CaCert)
	err = tp1.ConnectWithPassword(TestUser1Pass)
	require.NoError(t, err)
	defer tp1.Disconnect()
	time.Sleep(time.Millisecond)
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	defer slog.Info("--- TestLoginFail end")

	srv, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer srv.Stop()

	// add several users, service and devices
	err = srv.ApplyAuth(NatsTestClients)
	require.NoError(t, err)

	serverURL, _, _ := srv.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(serverURL, TestUser1ID, certBundle.CaCert)
	err = tp1.ConnectWithPassword("wrongpassword")
	require.Error(t, err)

	// key doesn't belong to user
	//hc1, err = natshubclient.ConnectWithNKey(
	//	serverURL, user1ID, TestService1NKey, certBundle.CaCert)
	//require.Error(t, err)
}

// test if the $events ingress stream captures events
func TestEventsStream(t *testing.T) {
	t.Log("---TestEventsStream start---")
	defer t.Log("---TestEventsStream end---")
	const eventMsg = "hello world"
	rxChan := make(chan string, 1)
	var err error

	// setup
	srv, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer srv.Stop()
	_ = cfg
	// the main service can access $JS
	// add devices that publish things, eg TestDevice1ID and TestService1ID
	err = srv.ApplyAuth(NatsTestClients)
	require.NoError(t, err)

	serverURL, _, _ := srv.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(serverURL, TestService1ID, certBundle.CaCert)
	err = tp1.ConnectWithKey(TestService1NKey)
	//nc1, err := srv.ConnectInProc("core-test", nil)
	//hc1 := natshubclient.NewNatsHubClient("", "core-test", nil, nil)
	//err = hc1.ConnectWithConn("", nc1)
	require.NoError(t, err)
	defer tp1.Disconnect()

	// the events stream must exist
	js := tp1.JS()
	si, err := js.StreamInfo(service.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := tp1.SubStream(service.EventsIntakeStreamName, false,
		func(msg *things.ThingValue) {
			slog.Info("received event", "event name", msg.Name)
			rxChan <- string(msg.Data)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// connect as the device and publish a things event
	tp2 := natstransport.NewNatsTransport(serverURL, TestDevice1ID, certBundle.CaCert)
	err = tp2.ConnectWithKey(TestDevice1NKey)
	require.NoError(t, err)
	defer tp2.Disconnect()
	addr2 := natstransport.MakeSubject(
		transports.MessageTypeEvent, "device1", TestThing1ID, "event1", TestDevice1ID)

	err = tp2.PubEvent(addr2, []byte(eventMsg))
	require.NoError(t, err)

	// read the events stream for
	si, err = tp2.JS().StreamInfo(service.EventsIntakeStreamName)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))

	// check the result
	rxMsg := <-rxChan
	assert.Equal(t, eventMsg, rxMsg)
}
