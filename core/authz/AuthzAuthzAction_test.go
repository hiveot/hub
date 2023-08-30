package authz_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"sync/atomic"
	"testing"
	"time"
)

// see AuthzPermissions_test for setup

// Test receiving an action
func TestSubActions(t *testing.T) {
	logrus.Infof("---TestAction1 start---")
	defer logrus.Infof("---TestAction1 end---")
	const action1ID = "action1"
	const actionMsg = "hello world"
	//device1Pub, _ := device1KP.PublicKey()
	var rxMsg string
	var rxCount atomic.Int32
	var err error

	// setup
	svc, stopFn := startTestAuthzService()
	require.NotNil(t, svc)
	defer stopFn()

	// add the group with thing and user
	err = svc.CreateGroup(group1ID, "g 1", 0)
	require.NoError(t, err)
	err = svc.AddSource(device1ID, thing1ID, group1ID)
	require.NoError(t, err)
	err = svc.AddUser(user1ID, auth.ClientRoleOperator, group1ID)
	require.NoError(t, err)

	// connect as a device and listen for action requests
	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()
	// devices can subscribe to actions directed to them
	sub2, err := hc2.SubActions(thing1ID, func(actionMsg *hubclient.ActionMessage) error {
		slog.Info("received action", "sender", actionMsg.ClientID, "actionID", actionMsg.ActionID)
		actionMsg.SendAck()
		rxCount.Add(1)
		rxMsg = string(actionMsg.Payload)
		return nil
	})
	require.NoError(t, err)
	defer sub2.Unsubscribe()

	// connect as a user and publish an action
	hc1, err := natshubclient.ConnectWithPassword(clientURL, user1ID, user1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()
	resp, err := hc1.PubAction(device1ID, thing1ID, action1ID, []byte(actionMsg))
	assert.NoError(t, err)
	_ = resp

	// allow background process to complete publishing
	time.Sleep(time.Millisecond * 1)

	assert.Equal(t, int32(1), rxCount.Load())
	assert.Equal(t, actionMsg, rxMsg)
}
