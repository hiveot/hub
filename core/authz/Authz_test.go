package authz_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/authz/authzadapter"
	"github.com/hiveot/hub/core/authz/authzclient"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/hiveot/hub/lib/logging"
)

// Test setup for authz, shared with AuthzManage_test and AuthzXyz_test

var testDir = path.Join(os.TempDir(), "test-authz")
var aclFilename = "authz.acl"
var aclFilePath = path.Join(testDir, aclFilename)
var serverCfg *natsserver.NatsServerConfig
var certBundle certs.TestCertBundle

// the following are set by the testmain
var clientURL string
var msgServer *natsserver.NatsNKeyServer

// run the test for different cores
var useCore = "nats" // nats vs mqtt

// Create a new authz service with empty acl list
// Returns the authz and authn services for use in testing
func startTestAuthzService() (svc authz.IAuthz, closeFn func()) {
	var authzAdpt authz.IAuthz
	var hc1 hubclient.IHubClient

	_ = os.Remove(aclFilePath)
	aclStore := authzservice.NewAuthzFileStore(aclFilePath)
	// setup for the authz service
	nc1, err := msgServer.ConnectInProc("authz", nil)
	hc1, _ = natshubclient.ConnectWithNC(nc1, "authz")
	if err != nil {
		panic("can't connect to server: " + err.Error())
	}
	authzAdpt = authzadapter.NewNatsAuthzAdapter(aclStore, msgServer)
	authzSvc := authzservice.NewAuthzService(aclStore, authzAdpt, hc1)
	err = authzSvc.Start()
	if err != nil {
		panic("failed to start authz service: " + err.Error())
	}

	//--- create a hub client for the authz management
	nc2, err := msgServer.ConnectInProc("authz-client", nil)
	if err != nil {
		panic("unable to connect authz client")
	}
	hc2, err := natshubclient.ConnectWithNC(nc2, "authz-client")
	if err != nil {
		panic("unable to connect authz client to JS")
	}
	authzMng := authzclient.NewAuthzClient(hc2)

	return authzMng, func() {
		hc2.Disconnect()
		authzSvc.Stop()
	}
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	clientURL, msgServer, certBundle, serverCfg, err = testenv.StartNatsTestServer()
	if err != nil {
		panic(err)
	}
	res := m.Run()

	msgServer.Stop()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Test the all group
func TestEventsStream(t *testing.T) {
	logrus.Infof("---TestEventsStream start---")
	defer logrus.Infof("---TestEventsStream end---")
	const device1ID = "device1"
	const thing1ID = "thing1"
	const service1ID = "service1"
	const eventMsg = "hello world"
	var rxMsg string
	var err error

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()
	_ = svc

	// add devices that publish things
	device1Key, _ := nkeys.CreateUser()
	device1Pub, _ := device1Key.PublicKey()
	err = msgServer.AddDevice(device1ID, device1Pub)
	require.NoError(t, err)

	// create the service that will subscribe to the event
	service1Key, _ := nkeys.CreateUser()
	service1Pub, _ := service1Key.PublicKey()
	err = msgServer.AddService(service1ID, service1Pub)
	require.NoError(t, err)
	hc1, err := natshubclient.ConnectWithNKey(clientURL, service1ID, service1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// the stream must exist
	si, err := hc1.JS().StreamInfo(authzadapter.EventsIntakeStreamName)
	require.NoError(t, err)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//

	// create the stream consumer and listen for events
	sub, err := hc1.SubGroup(authzadapter.EventsIntakeStreamName, false,
		func(msg *hubclient.EventMessage) {
			slog.Info("received event", "eventID", msg.EventID)
			rxMsg = string(msg.Payload)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// connect as the device and publish a thing event
	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()

	err = hc2.PubEvent(thing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)

	// read the events stream for
	si, err = hc1.JS().StreamInfo(authzadapter.EventsIntakeStreamName)
	slog.Info("stream $events:",
		slog.Uint64("count", si.State.Msgs),
		slog.Int("consumers", si.State.Consumers))
	//
	time.Sleep(time.Millisecond * 1000)

	// check the result
	assert.Equal(t, eventMsg, rxMsg)
}

// Test the all group
func TestAllGroup(t *testing.T) {
	logrus.Infof("---TestAllGroup start---")
	defer logrus.Infof("---TestAllGroup end---")
	const device1ID = "device1"
	const thing1ID = "thing1"
	const user1ID = "user1"
	const eventMsg = "hello world"
	var rxMsg string
	var rxCount atomic.Int32
	var err error
	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// the 'all' group must exist
	grp, err := svc.GetGroup(authz.AllGroupName)
	require.NoError(t, err)
	assert.Equal(t, authz.AllGroupName, grp.Name)

	// add devices that publish things
	device1Key, _ := nkeys.CreateUser()
	device1Pub, _ := device1Key.PublicKey()
	err = msgServer.AddDevice(device1ID, device1Pub)
	require.NoError(t, err)

	// add an all-group viewer
	// only all group members can view
	user1Key, _ := nkeys.CreateUser()
	user1Pub, _ := user1Key.PublicKey()
	err = msgServer.AddUser(device1ID, "", user1Pub)
	require.NoError(t, err)
	hc1, err := natshubclient.ConnectWithNKey(clientURL, user1ID, user1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// thing events should be received by all group subscribers
	// test with $events stream
	sub1, err := hc1.SubGroup(authz.AllGroupName, false, func(msg *hubclient.EventMessage) {
		slog.Info("received event",
			"id", msg.EventID, "publisher", msg.BindingID, "thing", msg.ThingID)
		rxMsg = string(msg.Payload)
		rxCount.Add(1)
	})
	assert.NoError(t, err)
	defer sub1.Unsubscribe()

	// connect as a device and publish a thing event
	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.PubEvent(thing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)
	assert.Equal(t, int32(0), rxCount.Load())

	// add the user to the all group
	err = svc.AddUser(authz.AllGroupName, authz.GroupRoleViewer, user1Pub)
	require.NoError(t, err)

	// publish a second event. should be received now
	err = hc2.PubEvent(thing1ID, "event2", []byte(eventMsg))
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, eventMsg, rxMsg)
	assert.Equal(t, int32(1), rxCount.Load())
}

// Test that devices have authorization to publish TDs and events
func TestDeviceAuthorization(t *testing.T) {
	logrus.Infof("---TestDeviceAuthorization---")
	const group1ID = "group1"
	const device1ID = "device1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// the all group must exist
	err := svc.AddThing(device1ID, authz.AllGroupName)
	require.NoError(t, err)
	err = svc.AddGroup(group1ID, -1)
	require.NoError(t, err)
	err = svc.AddThing(thingID1, group1ID)
	assert.NoError(t, err)
	err = svc.AddThing(thingID2, group1ID)
	assert.NoError(t, err)

	// this test makes no sense as devices have authz but are not in ACLs
	perms, err := svc.GetPermissions(device1ID, []string{thingID1})
	assert.NoError(t, err)
	thingPerm := perms[thingID1]
	assert.Contains(t, thingPerm, authz.PermPubEvents)
	assert.Contains(t, thingPerm, authz.PermReadActions)
	assert.NotContains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermReadEvents)
}

func TestManagerAuthorization(t *testing.T) {
	logrus.Infof("---TestManagerAuthorization---")
	const client1ID = "manager1"
	const group1ID = "group1"
	const thingID1 = "thing1"
	const thingID2 = "thing2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, 0)
	require.NoError(t, err)
	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// services can do whatever as a manager in the all group
	// the manager in the allgroup takes precedence over the operator role in group1
	_ = svc.SetUserRole(client1ID, authz.GroupRoleOperator, group1ID)
	_ = svc.SetUserRole(client1ID, authz.GroupRoleManager, authz.AllGroupName)
	perms, _ := svc.GetPermissions(client1ID, []string{thingID1})
	thingPerm := perms[thingID1]

	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.Contains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermReadActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
}

func TestOperatorAuthorization(t *testing.T) {
	logrus.Infof("---TestOperatorAuthorization---")
	const client1ID = "operator1"
	const deviceID = "device1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, 0)
	require.NoError(t, err)
	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)
	_ = svc.AddThing(deviceID, group1ID)
	_ = svc.AddUser(client1ID, authz.GroupRoleOperator, group1ID)

	// operators can readTD, readEvent, emitAction
	_ = svc.SetUserRole(client1ID, authz.GroupRoleOperator, group1ID)
	perms, _ := svc.GetPermissions(client1ID, []string{thingID1})
	thingPerm := perms[thingID1]

	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.Contains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
	assert.NotContains(t, thingPerm, authz.PermReadActions)
}

func TestViewerAuthorization(t *testing.T) {
	logrus.Infof("---TestViewerAuthorization---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, 0)
	require.NoError(t, err)
	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// viewers role can read TD
	_ = svc.AddUser(user1ID, authz.GroupRoleViewer, group1ID)
	perms, _ := svc.GetPermissions(user1ID, []string{thingID1})
	thingPerm := perms[thingID1]

	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.NotContains(t, thingPerm, authz.PermReadActions)
	assert.NotContains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
}

func TestNoAuthorization(t *testing.T) {
	logrus.Infof("---TestNoAuthorization---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.AddGroup(group1ID, 0)
	err := svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// viewers role can read TD
	_ = svc.AddUser(user1ID, "badrole", group1ID)
	perms, _ := svc.GetPermissions(user1ID, []string{thingID1})
	require.Equal(t, 1, len(perms), "expected one thing return")
	thingPerm := perms[thingID1]
	assert.Equal(t, 0, len(thingPerm), "expected no permissions for thing")
}

func TestClientPermissions(t *testing.T) {
	const user1ID = "user1"
	const group1ID = "group1"
	const group2ID = "group2"
	const group3ID = "group3"
	const thing1ID = "urn:thing1"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.AddGroup(group1ID, 0)
	_ = svc.AddGroup(group2ID, 0)
	_ = svc.AddGroup(group3ID, 0)
	_ = svc.AddThing(thing1ID, group1ID)
	_ = svc.AddThing(thing1ID, group2ID)
	_ = svc.AddThing(thing1ID, group3ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleViewer, group1ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleManager, group2ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleOperator, group3ID)

	// as a manager, permissions to read events and emit actions
	perms, err := svc.GetPermissions(user1ID, []string{thing1ID})
	assert.NoError(t, err)
	thingPerm := perms[thing1ID]
	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.Contains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
	assert.NotContains(t, thingPerm, authz.PermReadActions)

	// after removing the manager and operator roles, write property permissions no longer apply
	err = svc.RemoveClient(user1ID, group2ID)
	err = svc.RemoveClient(user1ID, group3ID)
	assert.NoError(t, err)
	perms, err = svc.GetPermissions(user1ID, []string{thing1ID})
	assert.NoError(t, err)
	thingPerm = perms[thing1ID]
	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.NotContains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
	assert.NotContains(t, thingPerm, authz.PermReadActions)
}
