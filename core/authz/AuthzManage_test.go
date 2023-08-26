package authz_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Authz manage tests
// This uses TestMain from AuthzPermissions_test

func TestAuthzServiceStartStop(t *testing.T) {
	logrus.Infof("---TestAuthzServiceStartStop---")
	svc, closeFn := startTestAuthzService()
	defer closeFn()
	assert.NotNil(t, svc)
}

func TestAuthzServiceBadStart(t *testing.T) {
	logrus.Infof("---TestAuthzServiceBadStart---")
	badAclFilePath := "/bad/aclstore/path"
	aclStore := authzservice.NewAuthzFileStore(badAclFilePath)

	// opening the acl store should fail
	err := aclStore.Open()
	require.Error(t, err)
	svc := authzservice.NewAuthzService(aclStore, nil)

	// service opening the acl store should fail
	err = svc.Start()
	svc.Stop()

	// missing store should not panic
	svc = authzservice.NewAuthzService(nil, nil)
	err = svc.Start()
	assert.Error(t, err)
}

// Test that devices have authorization to publish TDs and events
func TestCreateDeleteGroups(t *testing.T) {
	logrus.Infof("---TestCreateDeleteGroups---")
	const group1ID = "group1"

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	// the all group must exist
	ag, err := svc.GetGroup(authz.AllGroupID)
	require.NoError(t, err)
	assert.Equal(t, authz.AllGroupID, ag.ID)

	// add a new group
	err = svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	ag, err = svc.GetGroup(group1ID)
	require.NoError(t, err)
	assert.Equal(t, group1ID, ag.ID)

	// remove
	err = svc.DeleteGroup(group1ID)
	require.NoError(t, err)
}
func TestAddRemoveThings(t *testing.T) {
	const group1ID = "group1"
	const group2ID = "group2"
	const thingID1 = "thing1"
	const thingID2 = "thing2"
	const thingID3 = "thing3"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.AddGroup(group2ID, "group 2", 0)

	// group1 has thing1
	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)
	// group2 has thing1-3
	err = svc.AddThing(thingID1, group2ID)
	require.NoError(t, err)
	err = svc.AddThing(thingID2, group2ID)
	require.NoError(t, err)
	err = svc.AddThing(thingID3, group2ID)
	require.NoError(t, err)

	// check memberships
	group1, err := svc.GetGroup(group1ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(group1.MemberRoles))
	assert.Equal(t, 3, len(group2.MemberRoles))
	assert.Contains(t, group1.MemberRoles, thingID1)
	assert.NotContains(t, group1.MemberRoles, thingID2)
	assert.NotContains(t, group1.MemberRoles, thingID3)
	assert.Contains(t, group2.MemberRoles, thingID1)
	assert.Contains(t, group2.MemberRoles, thingID2)
	assert.Contains(t, group2.MemberRoles, thingID3)

	// remove thing
	err = svc.RemoveClient(thingID3, group2ID)
	require.NoError(t, err)
	group2b, err := svc.GetGroup(group2ID)
	assert.Contains(t, group2b.MemberRoles, thingID1)
	assert.Contains(t, group2b.MemberRoles, thingID2)
	assert.NotContains(t, group2b.MemberRoles, thingID3)
}
func TestAddRemoveServices(t *testing.T) {
	const group1ID = "group1"
	const group2ID = "group2"
	const serviceID1 = "service1"
	const serviceID2 = "service2"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.AddGroup(group2ID, "group 2", 0)

	// group1 has service1
	err = svc.AddService(serviceID1, group1ID)
	require.NoError(t, err)
	// group2 has service1-2
	err = svc.AddService(serviceID1, group2ID)
	require.NoError(t, err)
	err = svc.AddService(serviceID2, group2ID)
	require.NoError(t, err)

	// check memberships
	group1, err := svc.GetGroup(group1ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(group1.MemberRoles))
	assert.Equal(t, 2, len(group2.MemberRoles))
	assert.Contains(t, group1.MemberRoles, serviceID1)
	assert.NotContains(t, group1.MemberRoles, serviceID2)
	assert.Contains(t, group2.MemberRoles, serviceID1)
	assert.Contains(t, group2.MemberRoles, serviceID2)

	// remove service
	err = svc.RemoveClient(serviceID2, group2ID)
	require.NoError(t, err)
	group2b, err := svc.GetGroup(group2ID)
	assert.Contains(t, group2b.MemberRoles, serviceID1)
	assert.NotContains(t, group2b.MemberRoles, serviceID2)
}
func TestAddRemoveUsers(t *testing.T) {
	const group1ID = "group1"
	const group2ID = "group2"
	const userID1 = "user1"
	const userID2 = "user2"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	group1b, err := svc.GetGroup(group1ID)
	if len(group1b.MemberRoles) != 0 {
		slog.Error("where do these come from?")
		return
	}

	err = svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.AddGroup(group2ID, "group 2", 0)

	// group1 has user1
	err = svc.AddUser(userID1, authz.GroupRoleViewer, group1ID)
	require.NoError(t, err)
	// group2 has user1-2
	err = svc.AddUser(userID1, authz.GroupRoleManager, group2ID)
	require.NoError(t, err)
	err = svc.AddUser(userID2, authz.GroupRoleOperator, group2ID)
	require.NoError(t, err)

	// check memberships
	group1, err := svc.GetGroup(group1ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	if len(group1.MemberRoles) != 1 {
		slog.Error("incorrect member roles")
	}
	assert.Equal(t, 1, len(group1.MemberRoles))
	assert.Equal(t, 2, len(group2.MemberRoles))
	assert.Contains(t, group1.MemberRoles, userID1)
	assert.NotContains(t, group1.MemberRoles, userID2)
	assert.Contains(t, group2.MemberRoles, userID1)
	assert.Contains(t, group2.MemberRoles, userID2)

	// remove user
	err = svc.RemoveClient(userID2, group2ID)
	require.NoError(t, err)
	group2b, err := svc.GetGroup(group2ID)
	assert.Contains(t, group2b.MemberRoles, userID1)
	assert.NotContains(t, group2b.MemberRoles, userID2)
}
func TestGetGroups(t *testing.T) {
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

	err := svc.AddGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	err = svc.AddGroup(group2ID, "group 2", 0)
	require.NoError(t, err)
	err = svc.AddGroup(group3ID, "group 3", 0)
	require.NoError(t, err)

	err = svc.AddThing(thingID1, group1ID)
	require.NoError(t, err)

	_ = svc.AddThing(thingID1, group2ID)
	_ = svc.AddThing(thingID2, group2ID)
	_ = svc.AddThing(thingID3, group3ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleViewer, group1ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleViewer, group2ID)

	// 3+1 groups must exist (+all group)
	groups, err := svc.GetClientGroups("")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(groups))

	// group 2 has 3 members, 2 things and 1 user
	group, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, group2ID, group.ID)
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

	_ = svc.AddGroup(group1ID, "group 1", 0)
	_ = svc.AddGroup(group2ID, "group 2", 0)
	_ = svc.AddGroup(group3ID, "group 3", 0)
	// user1 is a member of 3 groups
	err := svc.AddUser(user1ID, authz.GroupRoleOperator, group1ID)
	assert.NoError(t, err)
	_ = svc.AddUser(user1ID, authz.GroupRoleOperator, group2ID)
	_ = svc.AddUser(user1ID, authz.GroupRoleOperator, group3ID)

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
