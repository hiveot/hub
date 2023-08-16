package authz_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/authz/authzclient"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/authz/natsauthz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"

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
func startTestAuthzService() (svc authz.IAuthz, closeFn func()) {
	var authzAppl authz.IAuthz
	var hc1 hubclient.IHubClient

	_ = os.Remove(aclFilePath)
	aclStore := authzservice.NewAuthzFileStore(aclFilePath)
	err := aclStore.Open()
	if err != nil {
		panic(err)
	}
	if useCore == "nats" {
		nc1, err := msgServer.ConnectInProc("authz", nil)
		hc1, _ = natshubclient.ConnectWithNC(nc1, "authz")
		if err != nil {
			panic("can't connect authz to server: " + err.Error())
		}
		authzJS, err := nc1.JetStream()
		if err != nil {
			panic("can't get jetstream api: " + err.Error())
		}
		authzAppl, err = natsauthz.NewNatsAuthzAppl(authzJS)

		if err != nil {
			panic("can't initialize nats authz binding: " + err.Error())
		}
	} else if useCore == "mqtt" {
		//hc = NewMqttHubClient()
		//authzAll = mqttauthz.NewMqttAuthzAppl(hc)
	}
	if err != nil {
		panic("unable to connect: " + err.Error())
	}
	authSvc := authzservice.NewAuthzService(aclStore, authzAppl, hc1)
	err = authSvc.Start()
	if err != nil {
		return nil, nil
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
	mngAuthz := authzclient.NewAuthzClient(hc2)

	return mngAuthz, func() {
		hc2.Disconnect()
		authSvc.Stop()
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
	err = svc.AddGroup(group1ID, 0)
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
