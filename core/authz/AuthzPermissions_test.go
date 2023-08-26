package authz_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authz/authzclient"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nkeys"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hub/lib/logging"
)

// Test setup for authz, shared with AuthzManage_test and AuthzXyz_test

var testDir = path.Join(os.TempDir(), "test-authz")
var aclFilename = "authz.acl"
var aclFilePath = path.Join(testDir, aclFilename)
var passwordFile = path.Join(testDir, "test.passwd")

// var serverCfg *natsserver.NatsServerConfig
var certBundle certs.TestCertBundle

// the following are set by the testmain
var clientURL string
var msgServer *natsnkeyserver.NatsNKeyServer
var serverCfg *natsnkeyserver.NatsServerConfig

var device1ID = "device1"
var device1Key, _ = nkeys.CreateUser()
var device1Pub, _ = device1Key.PublicKey()
var thing1ID = "thing1"

var user1ID = "user1"
var user1Pass = "pass1"
var user1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(user1Pass), 0)

var user2ID = "user2"
var user2Key, _ = nkeys.CreateUser()
var user2Pub, _ = user2Key.PublicKey()

var service1ID = "service1"
var service1Key, _ = nkeys.CreateUser()
var service1Pub, _ = service1Key.PublicKey()

var group1ID = "group1"
var group2ID = "group2"

// test users
var authzTestClients = []authn.AuthnEntry{
	{ClientProfile: authn.ClientProfile{
		ClientID:    device1ID,
		ClientType:  authn.ClientTypeDevice,
		DisplayName: "device1 1",
		PubKey:      device1Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    user1ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 1",
	}, PasswordHash: string(user1bcrypt),
	},
	{ClientProfile: authn.ClientProfile{
		ClientID:    user2ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 2",
		PubKey:      user2Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    service1ID,
		ClientType:  authn.ClientTypeService,
		DisplayName: "service 1",
		PubKey:      service1Pub,
	}},
}

// Create a new authz service with empty acl list
// Returns the authz and authn services for use in testing
func startTestAuthzService() (svc authz.IAuthz, closeFn func()) {
	cfg := authzservice.AuthzConfig{
		DataDir: "",
	}
	_ = cfg.Setup(testDir)
	if cfg.DataDir == "" {
		panic("missing data dir")
	}
	_ = os.RemoveAll(cfg.DataDir)
	authzSvc, err := authzservice.StartAuthzService(cfg, msgServer)
	if err != nil {
		panic("failed to start authz service: " + err.Error())
	}

	//--- create a hub client for the authz management
	hc, err := msgServer.ConnectInProc("authz-client")
	if err != nil {
		panic("unable to connect authz client")
	}
	authzMng := authzclient.NewAuthzClient(hc)

	return authzMng, func() {
		hc.Disconnect()
		authzSvc.Stop()
		time.Sleep(time.Millisecond * 100)
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
	msgServer.ApplyAuthn(authzTestClients)
	res := m.Run()

	msgServer.Stop()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Test that devices have permissions to publish TDs and events
func TestDevicePermissions(t *testing.T) {
	logrus.Infof("---TestDevicePermissions---")
	const group1ID = "group1"
	const device1ID = "device1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// the all group must exist
	err := svc.AddThing(device1ID, authz.AllGroupID)
	require.NoError(t, err)
	err = svc.AddGroup(group1ID, "group 1", -1)
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

func TestManagerPermissions(t *testing.T) {
	logrus.Infof("---TestManagerPermissions---")
	const client1ID = "manager1"
	const group1ID = "group1"
	const thingID1 = "thing1"
	const thingID2 = "thing2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// services can do whatever as a manager in the all group
	// the manager in the allgroup takes precedence over the operator role in group1
	_ = svc.SetUserRole(client1ID, authz.GroupRoleOperator, group1ID)
	_ = svc.SetUserRole(client1ID, authz.GroupRoleManager, authz.AllGroupID)
	perms, _ := svc.GetPermissions(client1ID, []string{thingID1})
	thingPerm := perms[thingID1]

	assert.Contains(t, thingPerm, authz.PermReadEvents)
	assert.Contains(t, thingPerm, authz.PermPubActions)
	assert.NotContains(t, thingPerm, authz.PermReadActions)
	assert.NotContains(t, thingPerm, authz.PermPubEvents)
}

func TestOperatorPermissions(t *testing.T) {
	logrus.Infof("---TestOperatorPermissions---")
	const client1ID = "operator1"
	const deviceID = "device1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, "group 1", 0)
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

func TestViewerPermissions(t *testing.T) {
	logrus.Infof("---TestViewerPermissions---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, "group 1", 0)
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

func TestNoPermission(t *testing.T) {
	logrus.Infof("---TestNoAuthorization---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "sensor1"
	const thingID2 = "sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.AddGroup(group1ID, "group 1", 0)
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

func TestThingPermissions(t *testing.T) {
	const user1ID = "user1"
	const group1ID = "group1"
	const group2ID = "group2"
	const group3ID = "group3"
	const thing1ID = "urn:thing1"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.AddGroup(group1ID, "group 1", 0)
	_ = svc.AddGroup(group2ID, "group 2", 0)
	_ = svc.AddGroup(group3ID, "group 3", 0)
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
