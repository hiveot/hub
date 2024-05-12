package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/lib/hubclient/httpclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"
)

const TestPort = 9444

var testDir = path.Join(os.TempDir(), "test-runtime")
var rtEnv = plugin.GetAppEnvironment(testDir, false)
var rtConfig = runtime.NewRuntimeConfig()
var certsBundle = certs.CreateTestCertBundle()

// start the runtime
func startRuntime() *runtime.Runtime {
	r := runtime.NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	if err != nil {
		panic("Failed to start runtime:" + err.Error())
	}
	return r
}

// Add a client and connect with its password
func addConnectClient(r *runtime.Runtime, clientType api.ClientType, clientID string) (
	cl hubclient.IHubClient, token string) {

	password := "pass1"
	err := r.AuthnSvc.AdminSvc.AddClient(clientType, clientID, clientID, "", password)
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	hostPort := fmt.Sprintf("localhost:%d", TestPort)
	cl = httpclient.NewHttpSSEClient(hostPort, clientID, certsBundle.CaCert)
	token, err = cl.ConnectWithPassword(password)

	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return cl, token
}

// create a TD with the given thingID
func createTD(thingID string) *things.TD {
	td := things.NewTD(thingID, "title-"+thingID, vocab.ThingSensor)
	return td
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	rtConfig.Protocols.HttpsBinding.Port = TestPort
	rtConfig.CaCert = certsBundle.CaCert
	rtConfig.CaKey = certsBundle.CaKey
	rtConfig.ServerKey = certsBundle.ServerKey
	rtConfig.ServerCert = certsBundle.ServerCert
	err := rtConfig.Setup(&rtEnv)
	if err != nil {
		panic("setup runtime config failed: " + err.Error())
	}

	_ = os.RemoveAll(testDir)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	r := startRuntime()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const senderID = "sender1"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)
	t2, err := cl.RefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, t2)

	cl.Disconnect()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestActionWithDeliveryConfirmation(t *testing.T) {
	const agentID = "agent1"
	const userID = "user1"
	const pass1 = "pass1"
	const thingID = "thing1"
	const actionID = "action1"
	const actionPayload = "payload1"
	var expectedReply = actionPayload + ".reply"
	var rxMsg *things.ThingMessage
	var stat3 api.DeliveryStatus

	r := startRuntime()
	err := r.AuthnSvc.AdminSvc.AddClient(api.ClientTypeAgent, agentID, "agent 1", "", "")
	require.NoError(t, err)
	err = r.AuthnSvc.AdminSvc.AddClient(api.ClientTypeUser, userID, "user 1", "", pass1)
	require.NoError(t, err)
	token1 := r.AuthnSvc.SessionAuth.CreateSessionToken(agentID, "", 100)

	// connect the agent and user clients
	// todo: iterate each protocol
	connectURL := r.ProtocolMgr.GetConnectURL()
	hc1 := connect.NewHubClient(connectURL, agentID, certsBundle.CaCert)
	newToken, err := hc1.ConnectWithJWT(token1)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	defer hc1.Disconnect()

	// Agent receives action request which we'll handle here
	hc1.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		rxMsg = msg
		stat.Reply = []byte(string(msg.Data) + ".reply")
		stat.Status = api.DeliveryCompleted
		return stat
	})

	// User publishes a request and receives delivery completion event
	hc2 := connect.NewHubClient(connectURL, userID, certsBundle.CaCert)
	newToken, err = hc2.ConnectWithPassword(pass1)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	defer hc2.Disconnect()

	// timeout is high for testing
	deliveryCtx, deliveryCtxComplete := context.WithTimeout(context.Background(), time.Minute*10)
	hc2.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		// client receives delivery updaes
		if msg.Key == vocab.EventTypeDeliveryUpdate {
			err = json.Unmarshal(msg.Data, &stat3)
			require.NoError(t, err)
			slog.Info(fmt.Sprintf("reply: %s", stat3.Reply))
		}
		stat.Status = api.DeliveryCompleted
		defer deliveryCtxComplete()
		return stat
	})
	// todo handle status update

	// agent publishes a TD and subscribes to action requests
	td1 := createTD(thingID)
	tdJSON, _ := json.Marshal(td1)
	stat := hc1.PubEvent(thingID, vocab.EventTypeTD, tdJSON)
	require.Equal(t, api.DeliveryCompleted, stat.Status)
	require.Empty(t, stat.Error)
	time.Sleep(time.Millisecond)

	// client sends action and expect a 'delivered' result
	dtThingID := things.MakeDigiTwinThingID(agentID, thingID)
	stat2 := hc2.PubAction(dtThingID, actionID, []byte(actionPayload))
	// TODO change this Completed once wait for completion is supported
	require.Equal(t, stat2.Status, api.DeliveryDelivered)
	require.Empty(t, stat2.Error)

	// wait for delivery completion
	select {
	case <-deliveryCtx.Done():
	}
	time.Sleep(time.Millisecond * 10)

	// verify final result
	require.Equal(t, stat2.MessageID, stat3.MessageID)
	require.Equal(t, stat3.Status, api.DeliveryCompleted)
	require.Empty(t, stat3.Error)
	require.NotNil(t, rxMsg)
	assert.Equal(t, expectedReply, string(stat3.Reply))
	assert.Equal(t, thingID, rxMsg.ThingID)
	assert.Equal(t, actionID, rxMsg.Key)
	assert.Equal(t, vocab.MessageTypeAction, rxMsg.MessageType)

}
