package natsnkeyserver_test

import (
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
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

func TestStartStopNKeysServer(t *testing.T) {
	var rxMsg string

	clientURL, s, _, _, err := testenv.StartNatsTestServer()

	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)
	//err = s.ApplyAuth(TestClients, TestRoles)
	//require.NoError(t, err)

	// connect using the built-in service key
	nc, err := s.ConnectInProcNC("testnkeysservice", nil)
	require.NoError(t, err)
	_, err = nc.Subscribe("things.>", func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	require.NoError(t, err)
	err = nc.Publish("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

	// make sure jetstream is enabled for account
	js, err := nc.JetStream()
	require.NoError(t, err)
	ai, err := js.AccountInfo()
	require.NoError(t, err)
	require.NotEmpty(t, ai)
	nc.Close()
}

func TestConnectWithNKey(t *testing.T) {

	slog.Info("--- TestConnectWithNKey start")
	defer slog.Info("--- TestConnectWithNKey end")
	var rxMsg string

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)

	// users subscribe to things
	hc1, err := natshubclient.ConnectWithNKey(
		clientURL, testenv.TestServiceID, testenv.TestService1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	subj1 := natshubclient.MakeThingsSubject("", "", natshubclient.MessageTypeEvent, "")
	_, err = hc1.Subscribe(subj1, func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	subj2 := natshubclient.MakeThingsSubject(testenv.TestServiceID, "thing1", "event", "test")
	err = hc1.Pub(subj2, []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

}
func TestConnectWithPassword(t *testing.T) {
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithPassword(
		clientURL, testenv.TestUser1ID, testenv.TestUser1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()
	time.Sleep(time.Millisecond)
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	defer slog.Info("--- TestLoginFail end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithPassword(
		clientURL, testenv.TestUser1ID, "wrongpassword", certBundle.CaCert)
	require.Error(t, err)

	// key doesn't belong to user
	//hc1, err = natshubclient.ConnectWithNKey(
	//	clientURL, user1ID, TestService1Key, certBundle.CaCert)
	//require.Error(t, err)

	_ = hc1
}

// test if the $events ingress stream captures events
func TestEventsStream(t *testing.T) {
	logrus.Infof("---TestEventsStream start---")
	defer logrus.Infof("---TestEventsStream end---")
	const eventMsg = "hello world"
	var rxMsg string
	var err error

	// setup
	clientURL, s, certBundle, cfg, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	_ = cfg
	// the main service can access $JS
	// add devices that publish things, eg TestDevice1ID and TestService1ID
	err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)

	//hc1, err := natshubclient.ConnectWithNKey(
	//	clientURL, testenv.TestService1ID, testenv.TestService1Key, certBundle.CaCert)
	nc1, err := s.ConnectInProcNC("core-test", nil)
	hc1, _ := natshubclient.ConnectWithNC(nc1)
	require.NoError(t, err)
	defer nc1.Close()

	// the events stream must exist
	js, _ := nc1.JetStream()
	si, err := js.StreamInfo(natsnkeyserver.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := hc1.SubStream(natsnkeyserver.EventsIntakeStreamName, false,
		func(msg *hubclient.EventMessage) {
			slog.Info("received event", "eventID", msg.EventID)
			rxMsg = string(msg.Payload)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// connect as the device and publish a thing event
	hc2, err := natshubclient.ConnectWithNKey(clientURL, testenv.TestDevice1ID, testenv.TestDevice1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()

	err = hc2.PubEvent(testenv.TestThing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)

	// read the events stream for
	si, err = hc1.JS().StreamInfo(natsnkeyserver.EventsIntakeStreamName)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//
	time.Sleep(time.Millisecond * 1000)

	// check the result
	assert.Equal(t, eventMsg, rxMsg)
}