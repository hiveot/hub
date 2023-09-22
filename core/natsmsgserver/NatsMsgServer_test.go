package natsmsgserver_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl/natshubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

const withCallout = false

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestStartStopNKeysServer(t *testing.T) {
	rxChan := make(chan string, 1)

	serverURL, s, _, _, err := testenv.StartNatsTestServer(withCallout)

	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, serverURL)
	//err = s.ApplyAuth(TestClients, TestRoles)
	//require.NoError(t, err)

	// connect using the built-in service key
	nc, err := s.ConnectInProcNC("testnkeysservice", nil)
	require.NoError(t, err)
	_, err = nc.Subscribe("things.>", func(msg *nats.Msg) {
		rxChan <- string(msg.Data)
		slog.Info("received message", slog.String("msg", string(msg.Data)))
	})
	require.NoError(t, err)
	err = nc.Publish("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	rxMsg := <-rxChan
	assert.Equal(t, "hello world", rxMsg)

	// make sure jetstream is enabled for account
	js, err := nc.JetStream()
	require.NoError(t, err)
	ai, err := js.AccountInfo()
	require.NoError(t, err)
	require.NotEmpty(t, ai)
	nc.Close()
}

func TestConnectWithCert(t *testing.T) {
	slog.Info("--- TestConnectWithCert start")
	defer slog.Info("--- TestConnectWithCert end")

	// this only works with callout
	serverURL, srv, _, certBundle, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer srv.Stop()

	// user1 used in this test must exist
	_ = srv.ApplyAuth(testenv.NatsTestClients)

	key, _ := certs.CreateECDSAKeys()
	clientCert, err := certs.CreateClientCert(testenv.TestUser1ID, auth.ClientRoleAdmin,
		1, &key.PublicKey, certBundle.CaCert, certBundle.CaKey)
	require.NoError(t, err)
	cl := natshubclient.NewNatsHubClient(serverURL, testenv.TestUser1ID, nil, certBundle.CaCert)
	clientTLS := certs.X509CertToTLS(clientCert, key)
	err = cl.ConnectWithCert(*clientTLS)
	assert.NoError(t, err)
	defer cl.Disconnect()
}

func TestConnectWithNKey(t *testing.T) {

	slog.Info("--- TestConnectWithNKey start")
	defer slog.Info("--- TestConnectWithNKey end")
	rxChan := make(chan string, 1)

	serverURL, s, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, serverURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.NatsTestClients)
	require.NoError(t, err)

	// users subscribe to things
	hc1 := natshubclient.NewNatsHubClient(serverURL, testenv.TestService1ID, testenv.TestService1NKey, certBundle.CaCert)
	err = hc1.ConnectWithKey()
	require.NoError(t, err)
	defer hc1.Disconnect()

	subj1 := natshubclient.MakeSubject(
		vocab.MessageTypeEvent, "", "", "", "")
	_, err = hc1.Sub(subj1, func(addr string, payload []byte) {
		rxChan <- string(payload)
		slog.Info("received message", "msg", string(payload))
	})
	assert.NoError(t, err)
	subj2 := natshubclient.MakeSubject(
		vocab.MessageTypeEvent, testenv.TestService1ID, "thing1", "test", "")
	err = hc1.Pub(subj2, []byte("hello world"))
	require.NoError(t, err)
	rxMsg := <-rxChan
	assert.Equal(t, "hello world", rxMsg)

}
func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	serverURL, s, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, serverURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.NatsTestClients)
	require.NoError(t, err)

	hc1 := natshubclient.NewNatsHubClient(serverURL, testenv.TestUser1ID, nil, certBundle.CaCert)
	err = hc1.ConnectWithPassword(testenv.TestUser1Pass)
	require.NoError(t, err)
	defer hc1.Disconnect()
	time.Sleep(time.Millisecond)
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	defer slog.Info("--- TestLoginFail end")

	serverURL, s, _, certBundle, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, serverURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.NatsTestClients)
	require.NoError(t, err)

	hc1 := natshubclient.NewNatsHubClient(serverURL, testenv.TestUser1ID, nil, certBundle.CaCert)
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
	serverURL, s, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	_ = cfg
	// the main service can access $JS
	// add devices that publish things, eg TestDevice1ID and TestService1ID
	err = s.ApplyAuth(testenv.NatsTestClients)
	require.NoError(t, err)

	//hc1, err := natshubclient.ConnectWithNKey(
	//	serverURL, testenv.TestService1ID, testenv.TestService1NKey, certBundle.CaCert)
	nc1, err := s.ConnectInProcNC("core-test", nil)
	hc1 := natshubclient.NewNatsHubClient("", "core-test", nil, nil)
	err = hc1.ConnectWithConn("", nc1)
	require.NoError(t, err)
	defer nc1.Close()

	// the events stream must exist
	js, _ := nc1.JetStream()
	si, err := js.StreamInfo(natsmsgserver.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := hc1.SubStream(natsmsgserver.EventsIntakeStreamName, false,
		func(msg *hubclient.EventMessage) {
			slog.Info("received event", "eventID", msg.EventID)
			rxChan <- string(msg.Payload)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// connect as the device and publish a thing event
	hc2 := natshubclient.NewNatsHubClient(serverURL, testenv.TestDevice1ID, testenv.TestDevice1NKey, certBundle.CaCert)
	err = hc2.ConnectWithKey()
	require.NoError(t, err)
	defer hc2.Disconnect()

	err = hc2.PubEvent(testenv.TestThing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)

	// read the events stream for
	si, err = hc1.JS().StreamInfo(natsmsgserver.EventsIntakeStreamName)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))

	// check the result
	rxMsg := <-rxChan
	assert.Equal(t, eventMsg, rxMsg)
}
