package natsmsgserver_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/natsmsgserver/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/natshubclient"
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
		ClientType: auth2.ClientTypeUser,
		PubKey:     TestAdminUserNPub,
		Role:       auth2.ClientRoleAdmin,
	},
	{
		ClientID:   TestDevice1ID,
		ClientType: auth2.ClientTypeDevice,
		PubKey:     TestDevice1NPub,
		Role:       auth2.ClientRoleDevice,
	},
	{
		ClientID:     TestUser1ID,
		ClientType:   auth2.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         auth2.ClientRoleViewer,
	},
	{
		ClientID:   TestService1ID,
		ClientType: auth2.ClientTypeService,
		PubKey:     TestService1NPub,
		Role:       auth2.ClientRoleService, // admin adds the $JS.API.INFO permissions
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
	//nc, err := srv.ConnectInProcNC("testnkeysservice", nil)
	serverURL, _, _ := srv.GetServerURLs()
	hc := natshubclient.NewNatsHubClient(serverURL, TestAdminUserID, TestAdminUserNKey, certBundle.CaCert)
	err = hc.ConnectWithKey()
	require.NoError(t, err)
	defer hc.Disconnect()

	// make sure jetstream is enabled for account
	js := hc.JS()
	ai, err := js.AccountInfo()
	require.NoError(t, err)
	require.NotEmpty(t, ai)
}

func TestConnectWithCert(t *testing.T) {
	slog.Info("--- TestConnectWithCert start")
	defer slog.Info("--- TestConnectWithCert end")

	// this only works with callout
	srv, _, certBundle, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer srv.Stop()

	// user1 used in this test must exist
	_ = srv.ApplyAuth(NatsTestClients)

	key, _ := certs.CreateECDSAKeys()
	clientCert, err := certs.CreateClientCert(TestUser1ID, auth2.ClientRoleAdmin,
		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
	require.NoError(t, err)
	serverURL, _, _ := srv.GetServerURLs()
	cl := natshubclient.NewNatsHubClient(serverURL, TestUser1ID, nil, certBundle.CaCert)
	clientTLS := certs.X509CertToTLS(clientCert, key)
	err = cl.ConnectWithCert(*clientTLS)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

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
	hc1 := natshubclient.NewNatsHubClient(serverURL, TestService1ID, TestService1NKey, certBundle.CaCert)
	err = hc1.ConnectWithKey()
	require.NoError(t, err)
	defer hc1.Disconnect()

	subSubj := natshubclient.MakeSubject(
		vocab.MessageTypeEvent, "", "", "", "")
	_, err = hc1.Sub(subSubj, func(addr string, payload []byte) {
		rxChan <- string(payload)
		slog.Info("received message", "msg", string(payload))
	})
	assert.NoError(t, err)
	pubSubj := natshubclient.MakeSubject(
		vocab.MessageTypeEvent, TestService1ID, "thing1", "test", TestService1ID)
	err = hc1.Pub(pubSubj, []byte("hello world"))
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
	hc1 := natshubclient.NewNatsHubClient(serverURL, TestUser1ID, nil, certBundle.CaCert)
	err = hc1.ConnectWithPassword(TestUser1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()
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
	hc1 := natshubclient.NewNatsHubClient(serverURL, TestUser1ID, nil, certBundle.CaCert)
	err = hc1.ConnectWithPassword("wrongpassword")
	require.Error(t, err)

	// key doesn't belong to user
	//hc1, err = natshubclient.ConnectWithNKey(
	//	serverURL, user1ID, TestService1NKey, certBundle.CaCert)
	//require.Error(t, err)

	_ = hc1
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
	hc1 := natshubclient.NewNatsHubClient(serverURL, TestService1ID, TestService1NKey, certBundle.CaCert)
	err = hc1.ConnectWithKey()
	//nc1, err := srv.ConnectInProcNC("core-test", nil)
	//hc1 := natshubclient.NewNatsHubClient("", "core-test", nil, nil)
	//err = hc1.ConnectWithConn("", nc1)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// the events stream must exist
	js := hc1.JS()
	si, err := js.StreamInfo(service.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := hc1.SubStream(service.EventsIntakeStreamName, false,
		func(msg *hubclient.EventMessage) {
			slog.Info("received event", "eventID", msg.EventID)
			rxChan <- string(msg.Payload)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// connect as the device and publish a thing event
	hc2 := natshubclient.NewNatsHubClient(serverURL, TestDevice1ID, TestDevice1NKey, certBundle.CaCert)
	err = hc2.ConnectWithKey()
	require.NoError(t, err)
	defer hc2.Disconnect()

	err = hc2.PubEvent(TestThing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)

	// read the events stream for
	si, err = hc1.JS().StreamInfo(service.EventsIntakeStreamName)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))

	// check the result
	rxMsg := <-rxChan
	assert.Equal(t, eventMsg, rxMsg)
}
