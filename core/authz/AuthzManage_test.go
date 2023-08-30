package authz_test

import (
	"github.com/hiveot/hub/api/go/auth"
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
	err = svc.CreateGroup(group1ID, "group 1", 0)
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

	err := svc.CreateGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.CreateGroup(group2ID, "group 2", 0)

	// group1 has thing1
	err = svc.AddSource(device1ID, thingID1, group1ID)
	require.NoError(t, err)
	// group2 has thing1-3
	err = svc.AddSource(device1ID, thingID1, group2ID)
	require.NoError(t, err)
	err = svc.AddSource(device1ID, thingID2, group2ID)
	require.NoError(t, err)
	err = svc.AddSource(device1ID, thingID3, group2ID)
	require.NoError(t, err)

	// check memberships
	group1, err := svc.GetGroup(group1ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(group1.Sources))
	assert.Equal(t, 3, len(group2.Sources))
	assert.Contains(t, group1.Sources, authz.EventSource{device1ID, thingID1})
	assert.NotContains(t, group1.Sources, authz.EventSource{device1ID, thingID2})
	assert.NotContains(t, group1.Sources, authz.EventSource{device1ID, thingID3})

	assert.Contains(t, group2.Sources, authz.EventSource{device1ID, thingID1})
	assert.Contains(t, group2.Sources, authz.EventSource{device1ID, thingID2})
	assert.Contains(t, group2.Sources, authz.EventSource{device1ID, thingID3})

	// remove thing
	err = svc.RemoveSource(device1ID, thingID3, group2ID)
	require.NoError(t, err)
	group2b, err := svc.GetGroup(group2ID)
	assert.Contains(t, group2b.Sources, authz.EventSource{device1ID, thingID1})
	assert.Contains(t, group2b.Sources, authz.EventSource{device1ID, thingID2})
	assert.NotContains(t, group2b.Sources, authz.EventSource{device1ID, thingID3})
}
func TestAddRemoveServices(t *testing.T) {
	const group1ID = "group1"
	const group2ID = "group2"
	const serviceID1 = "service1"
	const serviceID2 = "service2"
	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	err := svc.CreateGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.CreateGroup(group2ID, "group 2", 0)

	// group1 has service1
	err = svc.AddUser(serviceID1, authz.UserRoleService, group1ID)
	require.NoError(t, err)
	// group2 has service1-2
	err = svc.AddUser(serviceID1, authz.UserRoleService, group2ID)
	require.NoError(t, err)
	err = svc.AddUser(serviceID2, authz.UserRoleService, group2ID)
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
	err = svc.RemoveUser(serviceID2, group2ID)
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

	err = svc.CreateGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	_ = svc.CreateGroup(group2ID, "group 2", 0)

	// group1 has user1
	err = svc.AddUser(userID1, auth.ClientRoleViewer, group1ID)
	require.NoError(t, err)
	// group2 has user1-2
	err = svc.AddUser(userID1, auth.ClientRoleManager, group2ID)
	require.NoError(t, err)
	err = svc.AddUser(userID2, auth.ClientRoleOperator, group2ID)
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
	err = svc.RemoveUser(userID2, group2ID)
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

	err := svc.CreateGroup(group1ID, "group 1", 0)
	require.NoError(t, err)
	err = svc.CreateGroup(group2ID, "group 2", 0)
	require.NoError(t, err)
	err = svc.CreateGroup(group3ID, "group 3", 0)
	require.NoError(t, err)

	err = svc.AddSource(device1ID, thingID1, group1ID)
	require.NoError(t, err)

	_ = svc.AddSource(device1ID, thingID1, group2ID)
	_ = svc.AddSource(device1ID, thingID2, group2ID)
	_ = svc.AddSource(device1ID, thingID3, group3ID)
	_ = svc.AddUser(user1ID, auth.ClientRoleViewer, group1ID)
	_ = svc.AddUser(user1ID, auth.ClientRoleViewer, group2ID)

	// 3+1 groups must exist (+all group)
	groups, err := svc.GetUserGroups("")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(groups))

	// group 2 has 2 sources and 1 user
	group, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.Equal(t, group2ID, group.ID)
	assert.Equal(t, 1, len(group.MemberRoles))
	assert.Equal(t, 2, len(group.Sources))
	assert.Contains(t, group.MemberRoles, user1ID)
	assert.Contains(t, group.Sources,
		authz.EventSource{PublisherID: device1ID, ThingID: thingID1})
	assert.Contains(t, group.Sources,
		authz.EventSource{PublisherID: device1ID, ThingID: thingID2})

	// viewer1 is a member of 2 groups
	roles, err := svc.GetUserRoles(user1ID)
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

	// setup
	svc, stopFn := startTestAuthzService()
	defer stopFn()

	_ = svc.CreateGroup(group1ID, "group 1", 0)
	_ = svc.CreateGroup(group2ID, "group 2", 0)
	_ = svc.CreateGroup(group3ID, "group 3", 0)
	// user1 is a member of 3 groups
	err := svc.AddUser(user1ID, auth.ClientRoleOperator, group1ID)
	assert.NoError(t, err)
	_ = svc.AddUser(user1ID, auth.ClientRoleOperator, group2ID)
	_ = svc.AddUser(user1ID, auth.ClientRoleOperator, group3ID)

	// verify remove user1 from group 2
	err = svc.RemoveUser(user1ID, group2ID)
	assert.NoError(t, err)
	group2, err := svc.GetGroup(group2ID)
	assert.NoError(t, err)
	assert.NotContains(t, group2.MemberRoles, user1ID)

	roles, err := svc.GetUserRoles(user1ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(roles))
	assert.NotContains(t, roles, group2ID)

	// remove is idempotent.
	err = svc.RemoveUser(user1ID, group2ID)
	assert.NoError(t, err)
	// userID2 is not a member
	err = svc.RemoveUser(user2ID, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveUser(user2ID, group2ID)
	assert.NoError(t, err)
	err = svc.RemoveUser(user2ID, "notagroup")
	assert.Error(t, err)

	// removing all should remove user from all groups
	err = svc.RemoveUserAll(user1ID)
	roles, err = svc.GetUserRoles(user1ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(roles))
}
