package authz_test

import (
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/authz/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/hiveot/hub/lib/logging"
)

var testFolder = path.Join(os.TempDir(), "test-authz")
var testSocket = "authz.socket"
var aclFilename = "authz.acl"
var aclFilePath string
var authBundle testenv.TestAuthBundle
var tempFolder string

// Create a new authz service with empty acl list
func startTestAuthzService() (svc authz.IAuthz, closeFn func()) {

	_ = os.Remove(aclFilePath)
	aclStore := service.NewAuthzFileStore(aclFilePath, authz.AuthzServiceName)
	err := aclStore.Open()
	if err != nil {
		panic(err)
	}
	authSvc := service.NewAuthzService(aclStore, authBundle.CaCert)
	err = authSvc.Start()
	if err != nil {
		return nil, nil
	}

	return authSvc, func() {
		authSvc.Stop()
	}
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)
	aclFilePath = path.Join(testFolder, aclFilename)
	authBundle = testenv.CreateTestAuthBundle()

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(res)
}

func TestAuthzServiceStartStop(t *testing.T) {
	logrus.Infof("---TestAuthzServiceStartStop---")
	svc, closeFn := startTestAuthzService()
	defer closeFn()
	assert.NotNil(t, svc)
}

func TestAuthzServiceBadStart(t *testing.T) {
	logrus.Infof("---TestAuthzServiceBadStart---")
	badAclFilePath := "/bad/aclstore/path"
	aclStore := service.NewAuthzFileStore(badAclFilePath, authz.AuthzServiceName)

	// opening the acl store should fail
	err := aclStore.Open()
	require.Error(t, err)
	svc := service.NewAuthzService(aclStore, authBundle.CaCert)

	// service opening the acl store should fail
	err = svc.Start()
	svc.Stop()

	// missing store should not panic
	svc = service.NewAuthzService(nil, authBundle.CaCert)
	err = svc.Start()
	assert.Error(t, err)
}

// Test that devices have authorization to publish TDs and events
func TestDeviceAuthorization(t *testing.T) {
	logrus.Infof("---TestDeviceAuthorization---")
	const group1ID = "group1"
	const device1ID = "pub1"
	const thingID1 = "device1:sensor1"
	const thingID2 = "device2:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// FIXME: the device ID is normally not a member of the group
	err := svc.AddThing(device1ID, group1ID)
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
	const thingID1 = "device1:thing1"
	const thingID2 = "device1:thing2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// services can do whatever as a manager in the all group
	// the manager in the allgroup takes precedence over the operator role in group1
	_ = svc.SetUserRole(client1ID, authz.ClientRoleOperator, group1ID)
	_ = svc.SetUserRole(client1ID, authz.ClientRoleManager, authz.AllGroupName)
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
	const thingID1 = "device1:sensor1"
	const thingID2 = "device1:sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)
	_ = svc.AddThing(deviceID, group1ID)
	_ = svc.AddUser(client1ID, authz.ClientRoleOperator, group1ID)

	// operators can readTD, readEvent, emitAction
	_ = svc.SetUserRole(client1ID, authz.ClientRoleOperator, group1ID)
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
	const thingID1 = "device1:sensor1"
	const thingID2 = "device1:sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// viewers role can read TD
	_ = svc.AddUser(user1ID, authz.ClientRoleViewer, group1ID)
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
	const thingID1 = "device1:sensor1"
	const thingID2 = "device1:sensor2"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

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

func TestListGroups(t *testing.T) {
	const user1ID = "viewer1"
	const group1ID = "group1"
	const group2ID = "group2"
	const group3ID = "group3"
	const thingID1 = "device1:sensor1"
	const thingID2 = "device2:sensor2"
	const thingID3 = "device3:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)

	_ = svc.AddThing(thingID1, group2ID)
	_ = svc.AddThing(thingID2, group2ID)
	_ = svc.AddThing(thingID3, group3ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleViewer, group1ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleViewer, group2ID)

	// 3 groups must exist
	groups, err := svc.ListGroups("")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(groups))

	// group 2 has 3 members, 2 things and 1 user
	group, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, group2ID, group.Name)
	assert.Equal(t, 3, len(group.MemberRoles))
	assert.Contains(t, group.MemberRoles, thingID1)
	assert.Contains(t, group.MemberRoles, thingID2)
	assert.Contains(t, group.MemberRoles, user1ID)

	// viewer1 is a member of 2 groups
	roles, err := svc.GetClientRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(roles))
	assert.Contains(t, roles, group1ID)
	assert.Contains(t, roles, group2ID)

	// a non existing group has no members
	group, err = svc.GetGroup("notagroup")
	assert.Error(t, err)
}

func TestAddRemoveRoles(t *testing.T) {
	const user1ID = "user1op"
	const group1ID = "group1"
	const group2ID = "group2"
	const group3ID = "group3"
	const thingID1 = "urn:pub1:device1:sensor1"
	const thingID2 = "urn:pub2:device2:sensor2"
	//const thingID3 = "urn:pub2:device3:sensor1"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// user1 is a member of 3 groups
	err := svc.AddUser(user1ID, authz.ClientRoleOperator, group1ID)
	assert.NoError(t, err)
	_ = svc.AddUser(user1ID, authz.ClientRoleOperator, group2ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleOperator, group3ID)

	// thing1 is a member of 3 groups
	// adding a thing twice should not fail
	err = svc.AddThing(thingID1, group1ID)
	assert.NoError(t, err)
	_ = svc.AddThing(thingID1, group1ID)
	_ = svc.AddThing(thingID1, group2ID)
	_ = svc.AddThing(thingID1, group3ID)
	roles, err := svc.GetClientRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(roles))

	// verify remove thing1 from group 2
	err = svc.RemoveClient(thingID1, group2ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.NotContains(t, group2.MemberRoles, thingID1)

	roles, err = svc.GetClientRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(roles))
	assert.NotContains(t, roles, group2ID)

	// remove is idempotent.
	err = svc.RemoveClient(thingID1, group2ID)
	assert.NoError(t, err)
	// thingID2 is not a member
	err = svc.RemoveClient(thingID2, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveClient(thingID2, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveClient(thingID2, "notagroup")
	assert.Error(t, err)

	// removing all should remove user from all groups
	err = svc.RemoveClientAll(user1ID)
	roles, err = svc.GetClientRoles(user1ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(roles))
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

	_ = svc.AddThing(thing1ID, group1ID)
	_ = svc.AddThing(thing1ID, group2ID)
	_ = svc.AddThing(thing1ID, group3ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleViewer, group1ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleManager, group2ID)
	_ = svc.AddUser(user1ID, authz.ClientRoleOperator, group3ID)

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
