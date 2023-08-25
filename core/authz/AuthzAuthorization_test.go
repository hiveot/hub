package authz_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// see AuthzPermissions_test for setup

// Test the all group
func TestAllGroup(t *testing.T) {
	logrus.Infof("---TestAllGroup start---")
	defer logrus.Infof("---TestAllGroup end---")

	// The all group has all devices/things in it
	const eventMsg = "hello world"
	var rxMsg string
	var rxCount atomic.Int32
	var err error

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// the 'all' group must exist
	grp, err := svc.GetGroup(authz.AllGroupID)
	require.NoError(t, err)
	assert.Equal(t, authz.AllGroupID, grp.ID)

	// add an all-group viewer
	// only all group members can view
	hc1, err := natshubclient.ConnectWithPassword(clientURL, user1ID, user1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	// thing events should be received by 'all' group subscribers
	// FIXME - non members should not be able to subscribe
	sub1, err := hc1.SubGroup(authz.AllGroupID, false, func(msg *hubclient.EventMessage) {})
	assert.Error(t, err)

	// connect as a device and publish a thing event (device1 was already added to authn)
	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.PubEvent(thing1ID, "event1", []byte(eventMsg))
	require.NoError(t, err)
	// allow background process to complete publishing
	time.Sleep(time.Millisecond)
	assert.Equal(t, int32(0), rxCount.Load())

	// add the user to the all group
	err = svc.AddUser(user1ID, authz.GroupRoleViewer, authz.AllGroupID)

	hc1.Disconnect()
	hc1, err = natshubclient.ConnectWithPassword(clientURL, user1ID, user1Pass, certBundle.CaCert)
	// subscribe should work this time
	sub1, err = hc1.SubGroup(authz.AllGroupID, false, func(msg *hubclient.EventMessage) {
		slog.Info("received event",
			"id", msg.EventID, "publisher", msg.BindingID, "thing", msg.ThingID)
		rxMsg = string(msg.Payload)
		//rxMsg = string(msg.Data)
		rxCount.Add(1)
	})
	assert.NoError(t, err)
	defer sub1.Unsubscribe()

	// publish a second event. should be received now
	err = hc2.PubEvent(thing1ID, "event2", []byte(eventMsg))
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, eventMsg, rxMsg)
	assert.Equal(t, int32(1), rxCount.Load())
}
