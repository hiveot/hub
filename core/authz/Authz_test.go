package authz_test

import (
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authz/service"
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

var tempFolder string

// Create a new authz service with empty acl list
func startTestAuthzService() (svc hub.IAuthz, closeFn func()) {

	_ = os.Remove(aclFilePath)
	authSvc := service.NewAuthzService(aclFilePath)
	err := authSvc.Start()
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
	svc := service.NewAuthzService(badAclFilePath)

	// opening the acl store should fail
	err := svc.Start()
	assert.Error(t, err)
	svc.Stop()

	// missing store should not panic
	svc = service.NewAuthzService("")
	err = svc.Start()
	assert.Error(t, err)
}

// Test that devices have authorization to publish TDs and events
func TestDeviceAuthorization(t *testing.T) {
	logrus.Infof("---TestDeviceAuthorization---")
	const group1ID = "group1"
	const device1ID = "pub1"
	const thingID1 = "urn:zone1:pub1:device1:sensor1"
	const thingID2 = "urn:zone1:pub2:device2:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// FIXME: the device ID is normally not a member of the group
	err := svc.SetClientRole(device1ID, group1ID, hub.ClientRoleIotDevice)
	assert.NoError(t, err)
	err = svc.SetClientRole(thingID1, group1ID, hub.ClientRoleIotDevice)
	assert.NoError(t, err)
	err = svc.SetClientRole(thingID2, group1ID, hub.ClientRoleIotDevice)
	assert.NoError(t, err)

	// this test makes no sense as devices have authz but are not in ACLs
	perms, err := svc.GetPermissions(device1ID, thingID1)
	assert.NoError(t, err)
	assert.Contains(t, perms, hub.PermPubTD)
	assert.Contains(t, perms, hub.PermPubEvent)
	assert.Contains(t, perms, hub.PermReadAction)
	assert.NotContains(t, perms, hub.PermWriteProperty)
	assert.NotContains(t, perms, hub.PermEmitAction)
}

func TestManagerAuthorization(t *testing.T) {
	logrus.Infof("---TestManagerAuthorization---")
	const client1ID = "manager1"
	const group1ID = "group1"
	const thingID1 = "urn:zone1:pub1:device1:sensor1"
	const thingID2 = "urn:zone1:pub2:device1:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.SetClientRole(thingID1, group1ID, hub.ClientRoleIotDevice)
	_ = svc.SetClientRole(thingID2, group1ID, hub.ClientRoleIotDevice)

	// services can do whatever as a manager in the all group
	// the manager in the allgroup takes precedence over the operator role in group1
	_ = svc.SetClientRole(client1ID, group1ID, hub.ClientRoleOperator)
	_ = svc.SetClientRole(client1ID, hub.AllGroupName, hub.ClientRoleManager)
	perms, _ := svc.GetPermissions(client1ID, thingID1)

	assert.Contains(t, perms, hub.PermReadTD)
	assert.Contains(t, perms, hub.PermReadEvent)
	assert.Contains(t, perms, hub.PermEmitAction)
	assert.Contains(t, perms, hub.PermWriteProperty)
	assert.NotContains(t, perms, hub.PermPubTD)
	assert.NotContains(t, perms, hub.PermPubEvent)

}

func TestOperatorAuthorization(t *testing.T) {
	logrus.Infof("---TestOperatorAuthorization---")
	const client1ID = "operator1"
	const deviceID = "device1"
	const group1ID = "group1"
	const thingID1 = "urn:zone1:pub1:device1:sensor1"
	const thingID2 = "urn:zone1:pub2:device1:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.SetClientRole(thingID1, group1ID, hub.ClientRoleIotDevice)
	assert.NoError(t, err)
	_ = svc.SetClientRole(thingID2, group1ID, hub.ClientRoleIotDevice)
	_ = svc.SetClientRole(deviceID, group1ID, hub.ClientRoleIotDevice)
	_ = svc.SetClientRole(client1ID, group1ID, hub.ClientRoleOperator)

	// operators can readTD, readEvent, emitAction
	_ = svc.SetClientRole(client1ID, group1ID, hub.ClientRoleOperator)
	perms, _ := svc.GetPermissions(client1ID, thingID1)

	assert.Contains(t, perms, hub.PermReadTD)
	assert.Contains(t, perms, hub.PermReadEvent)
	assert.Contains(t, perms, hub.PermEmitAction)
	assert.NotContains(t, perms, hub.PermPubEvent)
	assert.NotContains(t, perms, hub.PermPubTD)
	assert.NotContains(t, perms, hub.PermWriteProperty)

}

func TestViewerAuthorization(t *testing.T) {
	logrus.Infof("---TestViewerAuthorization---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "urn:zone1:pub1:device1:sensor1"
	const thingID2 = "urn:zone1:pub2:device1:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.SetClientRole(thingID1, group1ID, hub.ClientRoleIotDevice)
	assert.NoError(t, err)
	_ = svc.SetClientRole(thingID2, group1ID, hub.ClientRoleIotDevice)

	// viewers role can read TD
	_ = svc.SetClientRole(user1ID, group1ID, hub.ClientRoleViewer)
	perms, _ := svc.GetPermissions(user1ID, thingID1)

	assert.Contains(t, perms, hub.PermReadTD)
	assert.Contains(t, perms, hub.PermReadEvent)
	assert.NotContains(t, perms, hub.PermEmitAction)
	assert.NotContains(t, perms, hub.PermPubEvent)
	assert.NotContains(t, perms, hub.PermPubTD)
	assert.NotContains(t, perms, hub.PermWriteProperty)
}

func TestNoAuthorization(t *testing.T) {
	logrus.Infof("---TestNoAuthorization---")
	const user1ID = "viewer1"
	const group1ID = "group1"
	const thingID1 = "urn:zone1:pub1:device1:sensor1"
	const thingID2 = "urn:zone1:pub2:device1:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	assert.NoError(t, err)
	_ = svc.AddThing(thingID2, group1ID)

	// viewers role can read TD
	_ = svc.SetClientRole(user1ID, group1ID, "badrole")
	perms, _ := svc.GetPermissions(user1ID, thingID1)
	assert.Equal(t, 0, len(perms))
}

func TestListGroups(t *testing.T) {
	const user1ID = "viewer1"
	const group1ID = "group1"
	const group2ID = "group2"
	const group3ID = "group3"
	const thingID1 = "urn:pub1:device1:sensor1"
	const thingID2 = "urn:pub2:device2:sensor2"
	const thingID3 = "urn:pub2:device3:sensor1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddThing(thingID1, group1ID)
	assert.NoError(t, err)

	_ = svc.AddThing(thingID1, group2ID)
	_ = svc.AddThing(thingID2, group2ID)
	_ = svc.AddThing(thingID3, group3ID)
	_ = svc.SetClientRole(user1ID, group1ID, hub.ClientRoleViewer)
	_ = svc.SetClientRole(user1ID, group2ID, hub.ClientRoleViewer)

	// 3 groups must exist
	groups, err := svc.ListGroups(0, 0)
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
	roles, err := svc.GetGroupRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(roles))
	assert.Contains(t, roles, group1ID)
	assert.Contains(t, roles, group2ID)

	// a non existing group has no members
	group, err = svc.GetGroup("notagroup")
	assert.Error(t, err)
}

func TestAddRemoveRoles(t *testing.T) {
	const user1ID = "viewer1"
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
	err := svc.SetClientRole(user1ID, group1ID, hub.ClientRoleOperator)
	assert.NoError(t, err)
	_ = svc.SetClientRole(user1ID, group2ID, hub.ClientRoleOperator)
	_ = svc.SetClientRole(user1ID, group3ID, hub.ClientRoleOperator)

	// thing1 is a member of 3 groups
	// adding a thing twice should not fail
	err = svc.AddThing(thingID1, group1ID)
	assert.NoError(t, err)
	_ = svc.AddThing(thingID1, group1ID)
	_ = svc.AddThing(thingID1, group2ID)
	_ = svc.AddThing(thingID1, group3ID)
	roles, err := svc.GetGroupRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(roles))

	// verify remove thing1 from group 2
	err = svc.RemoveThing(thingID1, group2ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.NotContains(t, group2.MemberRoles, thingID1)

	roles, err = svc.GetGroupRoles(thingID1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(roles))
	assert.NotContains(t, roles, group2ID)

	// remove is idempotent.
	err = svc.RemoveThing(thingID1, group2ID)
	assert.NoError(t, err)
	// thingID2 is not a member
	err = svc.RemoveThing(thingID2, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveClient(thingID2, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveClient(thingID2, "notagroup")
	assert.Error(t, err)

	// removing all should remove user from all groups
	err = svc.RemoveAll(user1ID)
	roles, err = svc.GetGroupRoles(user1ID)
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
	_ = svc.SetClientRole(user1ID, group1ID, hub.ClientRoleViewer)
	_ = svc.SetClientRole(user1ID, group2ID, hub.ClientRoleManager)
	_ = svc.SetClientRole(user1ID, group3ID, hub.ClientRoleOperator)

	// as a manager, permissions to read and emit actions
	perms, err := svc.GetPermissions(thing1ID, "")
	assert.NoError(t, err)
	assert.Contains(t, perms, hub.PermEmitAction)
	assert.Contains(t, perms, hub.PermWriteProperty)

	// after removing the manager role write property permissions no longer apply
	err = svc.RemoveThing(user1ID, group2ID)
	assert.NoError(t, err)
	perms, err = svc.GetPermissions(thing1ID, "")
	assert.NoError(t, err)
	assert.Contains(t, perms, hub.PermEmitAction)
	assert.NotContains(t, perms, hub.PermWriteProperty)

}
