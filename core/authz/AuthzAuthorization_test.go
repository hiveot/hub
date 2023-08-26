package authz_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// see AuthzPermissions_test for setup

// testing method to see if a user receives an event in a group
func testSub(t *testing.T, shouldFail bool, groupID string, userID string, pass string, deviceID string, deviceKey nkeys.KeyPair, thingID string) {

	const eventMsg = "hello world"
	var rxMsg string
	var rxCount atomic.Int32
	var err error

	// add the group with thing and user
	hc1, err := natshubclient.ConnectWithPassword(clientURL, userID, pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	sub1, err := hc1.SubGroup(groupID, false, func(msg *hubclient.EventMessage) {
		slog.Info("received event",
			"id", msg.EventID, "publisher", msg.BindingID, "thing", msg.ThingID)
		rxMsg = string(msg.Payload)
		rxCount.Add(1)
	})
	// to catch a failure, don't use assert here
	if err == nil {
		defer sub1.Unsubscribe()

		// connect as a device and publish a thing event (device1 was already added to authn)
		hc2, err := natshubclient.ConnectWithNKey(clientURL, deviceID, deviceKey, certBundle.CaCert)
		require.NoError(t, err)
		defer hc2.Disconnect()
		err = hc2.PubEvent(thingID, "event1", []byte(eventMsg))
		require.NoError(t, err)
		// allow background process to complete publishing
		time.Sleep(time.Millisecond * 1)
	}
	if shouldFail {
		assert.NotEqual(t, int32(1), rxCount.Load())
		assert.NotEqual(t, eventMsg, rxMsg)
	} else {
		assert.Equal(t, int32(1), rxCount.Load())
		assert.Equal(t, eventMsg, rxMsg)
	}
}

// Test the all group
func TestAllGroup(t *testing.T) {
	logrus.Infof("---TestAllGroup start---")
	defer logrus.Infof("---TestAllGroup end---")

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// the 'all' group must exist
	grp, err := svc.GetGroup(authz.AllGroupID)
	require.NoError(t, err)
	assert.Equal(t, authz.AllGroupID, grp.ID)

	// add an all-group viewer (user was already created in startTestAuthzService)
	// devices are implicitly included
	err = svc.AddUser(user1ID, authz.GroupRoleViewer, authz.AllGroupID)

	testSub(t, false, authz.AllGroupID, user1ID, user1Pass, device1ID, device1Key, thing1ID)
}

// Test a subscription group
func TestGroup1(t *testing.T) {
	logrus.Infof("---TestGroup1 start---")
	defer logrus.Infof("---TestGroup1 end---")

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// add the group with thing and user
	err := svc.AddGroup(group1ID, "g 1", 0)
	require.NoError(t, err)
	err = svc.AddThing(thing1ID, group1ID)
	require.NoError(t, err)
	err = svc.AddUser(user1ID, authz.GroupRoleViewer, group1ID)
	require.NoError(t, err)

	testSub(t, false, group1ID, user1ID, user1Pass, device1ID, device1Key, thing1ID)
}

// Test a subscription group
func TestGroup1NoThing(t *testing.T) {
	logrus.Infof("---TestGroup1 start---")
	defer logrus.Infof("---TestGroup1 end---")

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// add the group with thing and user
	err := svc.AddGroup(group1ID, "g 1", 0)
	require.NoError(t, err)
	err = svc.AddThing(thing1ID, group1ID)
	require.NoError(t, err)
	err = svc.AddUser(user1ID, authz.GroupRoleViewer, group1ID)
	require.NoError(t, err)

	// this should fail
	testSub(t, true, group1ID, user1ID, user1Pass, device1ID, device1Key, "thing2")
}

// Test a subscription group
func TestGroup2NotAMember(t *testing.T) {
	logrus.Infof("---TestGroup2NotAMember start---")
	defer logrus.Infof("---TestGroup2NotAMember end---")

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// add the group with thing and user
	err := svc.AddGroup(group1ID, "g 1", 0)
	require.NoError(t, err)
	err = svc.AddThing(thing1ID, group1ID)
	require.NoError(t, err)
	//err = svc.AddUser(user1ID, authz.GroupRoleViewer, group2ID) // group2, not group1
	require.NoError(t, err)

	// this should fail
	testSub(t, true, group1ID, user1ID, user1Pass, device1ID, device1Key, thing1ID)
}
