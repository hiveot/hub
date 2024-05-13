package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/inbox"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHttpsGetActions(t *testing.T) {
	const agThing1ID = "thing1"
	const key1 = "key1"
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"
	var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	// agent receives actions and sends events
	cl1, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer cl1.Disconnect()
	// consumer sends actions and receives events
	cl2, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl2.Disconnect()

	// consumer publish an action to the agent
	stat := cl2.PubAction(dtThing1ID, key1, []byte(data))
	require.Empty(t, stat.Error)

	// read the latest actions from the digitwin inbox
	args := inbox.ReadLatestArgs{ThingID: dtThing1ID}
	resp := inbox.ReadLatestResp{}
	ctx, rpcCancel := context.WithTimeout(context.Background(), time.Minute)
	stat2, err := cl2.Rpc(ctx, inbox.ThingID, inbox.ReadLatestMethod, &args, &resp)
	require.NoError(t, err)
	require.NotNil(t, stat2.Reply)
	rpcCancel()

	// the status result reply should also hold the data
	msg := inbox.ReadLatestResp{}
	//err = json.Unmarshal(resp.ThingValues, &msg)
	err = json.Unmarshal(stat2.Reply, &msg)
	require.NoError(t, err)
	actionMsg := msg.ThingValues[key1]
	require.NotNil(t, actionMsg)
	assert.Equal(t, data, string(actionMsg.Data))
}

// Get events from the outbox using the experimental http REST api
func TestHttpsGetEvents(t *testing.T) {
	const agentID = "agent1"
	const agThingID = "thing1"
	const key1 = "key1"
	const userID = "user1"
	const data = "Hello world"
	var dtThingID = things.MakeDigiTwinThingID(agentID, agThingID)

	r := startRuntime()
	defer r.Stop()
	// agent publishes events
	cl1, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer cl1.Disconnect()

	_ = cl1.PubEvent(agThingID, key1, []byte(data))

	// consumer reads
	cl, token := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl.Disconnect()

	hostPort := fmt.Sprintf("localhost:%d", TestPort)
	tlsClient := tlsclient.NewTLSClient(hostPort, certsBundle.CaCert, time.Minute)
	tlsClient.ConnectWithToken(userID, token)

	// read latest using the http API
	vars := map[string]string{"thingID": dtThingID}
	eventPath := utils.Substitute(vocab.GetEventsPath, vars)
	reply, err := tlsClient.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, reply)
	tmm := things.ThingMessageMap{}
	err = json.Unmarshal(reply, &tmm)
	require.NoError(t, err)
	require.NotZero(t, len(tmm))

	// read latest using the rest API
	args := outbox.ReadLatestArgs{ThingID: dtThingID}
	resp, stat, err := outbox.ReadLatest(cl.Rpc, args)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, stat.Error)
	require.NotEmpty(t, stat.Reply)

}
