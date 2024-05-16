package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/api"
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
	ts = testenv.NewTestServer()
	err := ts.Start(true)
	if err != nil {
		panic("Failed to start runtime:" + err.Error())
	}
	return ts.Runtime
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(ts.TestDir)
	}
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
	cl, _ := ts.AddConnectClient(api.ClientTypeUser, clientID, api.ClientRoleManager)
	t2, err := cl.RefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, t2)

	cl.Disconnect()
	r.Stop()
	//time.Sleep(time.Millisecond * 100)
}

func TestActionWithDeliveryConfirmation(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const pass1 = "pass1"
	const thingID = "thing1"
	const actionID = "action1"
	var actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"
	var rxMsg *things.ThingMessage
	var stat3 api.DeliveryStatus

	r := startRuntime()
	defer r.Stop()
	cl1, _ := ts.AddConnectClient(api.ClientTypeAgent, agentID, api.ClientRoleAgent)
	cl2, _ := ts.AddConnectClient(api.ClientTypeUser, userID, api.ClientRoleManager)

	// connect the agent and user clients
	defer cl1.Disconnect()
	defer cl2.Disconnect()

	// Agent receives action request which we'll handle here
	cl1.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		rxMsg = msg
		stat.Completed(msg, nil)
		//stat.Failed(msg, fmt.Errorf("failuretest"))
		stat.Reply = []byte(string(msg.Data) + ".reply")
		slog.Info("agent1 delivery complete", "messageID", msg.MessageID)
		return stat
	})

	// users receives delivery updates when sending actions
	deliveryCtx, deliveryCtxComplete := context.WithTimeout(context.Background(), time.Minute*10)
	cl2.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		if msg.Key == vocab.EventTypeDeliveryUpdate {
			// delivery updates are only invoked on for non-rpc actions
			err := json.Unmarshal(msg.Data, &stat3)
			require.NoError(t, err)
			slog.Info(fmt.Sprintf("reply: %s", stat3.Reply))
		}
		stat.Status = api.DeliveryCompleted
		defer deliveryCtxComplete()
		return stat
	})

	// client sends action to agent and expect a 'delivered' result
	// The RPC method returns an error if no reply is received
	dtThingID := things.MakeDigiTwinThingID(agentID, thingID)
	stat2, err := cl2.PubAction(dtThingID, actionID, []byte(actionPayload))
	_ = stat2
	require.NoError(t, err)

	// wait for delivery completion
	select {
	case <-deliveryCtx.Done():
	}
	time.Sleep(time.Millisecond * 10)

	// verify final result
	require.Equal(t, api.DeliveryCompleted, stat3.Status)
	require.Empty(t, stat3.Error)
	require.NotNil(t, rxMsg)
	assert.Equal(t, expectedReply, string(stat3.Reply))
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, actionID, rxMsg.Key)
	assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)

}
