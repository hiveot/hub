package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

var ts *testenv.TestServer

// start the test runtime
func startRuntime() *runtime.Runtime {
	ts = testenv.StartTestServer(true)
	return ts.Runtime
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	r := startRuntime()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const clientID = "user1"

	r := startRuntime()
	cl, token := ts.AddConnectUser(clientID, authn.ClientRoleManager)
	_ = token
	t2, err := cl.RefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, t2)
	// use the refresh token
	t3, err := cl.RefreshToken(t2)
	_ = t3
	require.NoError(t, err)

	cl.Disconnect()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestActionWithDeliveryConfirmation(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const thingID = "thing1"
	const actionID = "action1"
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"
	var rxMsg *things.ThingMessage
	var stat3 hubclient.DeliveryStatus

	r := startRuntime()
	defer r.Stop()
	cl1, _ := ts.AddConnectAgent(agentID)
	cl2, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)

	// connect the agent and user clients
	defer cl1.Disconnect()
	defer cl2.Disconnect()

	// Agent receives action request which we'll handle here
	cl1.SetActionHandler(func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		rxMsg = msg
		stat.Completed(msg, nil)
		//stat.Failed(msg, fmt.Errorf("failuretest"))
		stat.Reply = []byte(msg.DataAsText() + ".reply")
		slog.Info("agent1 delivery complete", "messageID", msg.MessageID)
		return stat
	})

	// users receives delivery updates when sending actions
	deliveryCtx, deliveryCtxComplete := context.WithTimeout(context.Background(), time.Minute*10)
	cl2.SetEventHandler(func(msg *things.ThingMessage) (err error) {
		if msg.Key == vocab.EventTypeDeliveryUpdate {
			// delivery updates are only invoked on for non-rpc actions
			err = msg.Unmarshal(&stat3)
			require.NoError(t, err)
			slog.Info(fmt.Sprintf("reply: %s", stat3.Reply))
		}
		defer deliveryCtxComplete()
		return err
	})

	// client sends action to agent and expect a 'delivered' result
	// The RPC method returns an error if no reply is received
	dThingID := things.MakeDigiTwinThingID(agentID, thingID)
	stat2 := cl2.PubAction(dThingID, actionID, []byte(actionPayload))
	require.Empty(t, stat2.Error)

	// wait for delivery completion
	select {
	case <-deliveryCtx.Done():
	}
	time.Sleep(time.Millisecond * 10)

	// verify final result
	require.Equal(t, hubclient.DeliveryCompleted, stat3.Status)
	require.Empty(t, stat3.Error)
	require.NotNil(t, rxMsg)
	assert.Equal(t, expectedReply, string(stat3.Reply))
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, actionID, rxMsg.Key)
	assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)

}

// Services and agents should auto-reconnect when server is restarted
func TestServiceReconnect(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const thingID = "thing1"
	const actionID = "action1"
	var rxMsg *things.ThingMessage
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"

	r := startRuntime()

	// give server time to start up before connecting
	time.Sleep(time.Millisecond * 10)

	cl1, cl1Token := ts.AddConnectAgent(agentID)
	defer cl1.Disconnect()

	// Agent receives action request which we'll handle here
	cl1.SetActionHandler(func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		var req string
		rxMsg = msg
		stat.Completed(msg, nil)
		_ = msg.Unmarshal(&req)
		stat.Reply, _ = json.Marshal(req + ".reply")
		slog.Info("agent1 delivery complete", "messageID", msg.MessageID)
		return stat
	})

	// give connection time to be established before stopping the server
	time.Sleep(time.Millisecond * 10)

	// after restarting the server, cl1's connection should automatically be re-established
	// TBD what is the go-sse reconnect algorithm? How to know it triggered?
	r.Stop()
	time.Sleep(time.Millisecond * 10)
	err := r.Start(&ts.AppEnv)
	require.NoError(t, err)
	defer r.Stop()

	cl2, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl2.Disconnect()
	// FIXME: detect a reconnect
	time.Sleep(time.Second * 5)

	// FIXME. this should not be needed
	_, err = cl1.ConnectWithToken(cl1Token)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)

	// this rpc call succeeds after agent1 has automatically reconnected
	dThingID := things.MakeDigiTwinThingID(agentID, thingID)
	var reply string
	err = cl2.Rpc(dThingID, actionID, &actionPayload, &reply)

	require.NoError(t, err, "auto-reconnect didn't take place")
	require.NotNil(t, rxMsg)
	require.Equal(t, expectedReply, string(reply))
}

// test that regular users don't have admin access to authn, authz
func TestAccess(t *testing.T) {
	const clientID = "user1"

	r := startRuntime()
	defer r.Stop()

	hc, token := ts.AddConnectUser(clientID, authn.ClientRoleViewer)
	defer hc.Disconnect()
	_ = token

	// regulars users should not have authn and authz admin access
	prof, err := authn.AdminGetProfiles(hc)
	require.Error(t, err, "regular users should not have access to authn.Admin")
	require.Empty(t, prof)
	//time.Sleep(time.Millisecond * 100)

	role, err := authz.AdminGetClientRole(hc, clientID)
	require.Error(t, err, "regular users should not have access to authz.Admin")
	require.Empty(t, role)
	//time.Sleep(time.Millisecond * 100)
}
