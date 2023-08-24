package natsnkeyserver_test

import (
	"github.com/hiveot/hub/api/go/authz"
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
	err = s.ApplyAuthn(testenv.TestClients)
	require.NoError(t, err)
	err = s.ApplyAuthz(testenv.TestRoles)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithNKey(
		clientURL, testenv.TestUser2ID, testenv.TestUser2Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Subscribe("things.>", func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

}
func TestConnectWithPassword(t *testing.T) {
	var rxMsg string
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ApplyAuthn(testenv.TestClients)
	require.NoError(t, err)
	err = s.ApplyAuthz(testenv.TestRoles)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithPassword(
		clientURL, testenv.TestUser1ID, testenv.TestUser1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Subscribe("things.>", func(msg *nats.Msg) {
		rxMsg = string(msg.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)
}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	defer slog.Info("--- TestLoginFail end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ApplyAuthn(testenv.TestClients)
	require.NoError(t, err)
	err = s.ApplyAuthz(testenv.TestRoles)
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
	clientURL, msgServer, certBundle, cfg, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer msgServer.Stop()
	_ = cfg
	// add devices that publish things, eg TestDevice1ID and TestService1ID
	err = msgServer.ApplyAuthn(testenv.TestClients)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithNKey(
		clientURL, testenv.TestService1ID, testenv.TestService1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// the events stream must exist
	si, err := hc1.JS().StreamInfo(natsnkeyserver.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := hc1.SubGroup(natsnkeyserver.EventsIntakeStreamName, false,
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

// test if a group with a device and user receives events
func TestAddGroup(t *testing.T) {
	logrus.Infof("---TestAddGroup start---")
	defer logrus.Infof("---TestAddGroup end---")
	const eventMsg = "hello world"
	var rxMsg string
	var err error

	var TestGroups = []authz.Group{
		{
			ID:          testenv.TestGroup1ID,
			DisplayName: "group 1",
			MemberRoles: authz.RoleMap{
				testenv.TestThing1ID: authz.GroupRoleThing,
				testenv.TestUser1ID:  authz.GroupRoleViewer,
			},
		},
	}
	// setup, add devices and a group
	clientURL, msgServer, certBundle, _, err := testenv.StartNatsTestServer()
	// add devices that publish things, eg TestDevice1ID and TestService1ID
	err = msgServer.ApplyAuthn(testenv.TestClients)
	assert.NoError(t, err)
	err = msgServer.ApplyGroups(TestGroups)
	assert.NoError(t, err)

	// setup user1 to receive events
	require.NoError(t, err)
	hc1, err := natshubclient.ConnectWithNKey(clientURL, testenv.TestService1ID, testenv.TestService1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	si1, _ := hc1.JS().StreamInfo(testenv.TestGroup1ID)
	_ = si1

	sub, err := hc1.SubGroup(testenv.TestGroup1ID, false,
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

	// thing2 should not be received
	err = hc2.PubEvent(testenv.TestThing2ID, "event2", []byte("thing2 message should not be received"))

	// give background processes time to receive and handle events
	time.Sleep(time.Millisecond * 1)

	// user1 should have received the event
	assert.Equal(t, eventMsg, rxMsg)
}
