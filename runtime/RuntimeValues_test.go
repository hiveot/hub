package runtime_test

import (
	"context"
	"encoding/json"
	"github.com/hiveot/hub/api/go/inbox"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHttpsGetActions(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()
	// agent receives actions and user sends actions
	cl1, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer cl1.Disconnect()
	cl2, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl2.Disconnect()

	// publish an action and read it
	stat := cl2.PubAction(thingID, key1, []byte(data))
	require.Empty(t, stat.Error)

	args := inbox.ReadLatestArgs{ThingID: thingID}
	resp := inbox.ReadLatestResp{}
	ctx, rpcCancel := context.WithTimeout(context.Background(), time.Minute)
	stat2, err := cl2.Rpc(ctx, thingID, inbox.ReadLatestMethod, &args, &resp)
	require.NoError(t, err)
	require.NotNil(t, stat2.Reply)
	rpcCancel()

	// the status result reply should also hold the data
	msg := things.ThingMessageMap{}
	err = json.Unmarshal(stat2.Reply, &msg)
	require.NoError(t, err)
	assert.Equal(t, data, string(msg[key1].Data))
}

func TestHttpsGetEvents(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const data = "Hello world"

	r := startRuntime()
	defer r.Stop()

	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)
	defer cl.Disconnect()

	//vars := map[string]string{"thingID": thingID}
	//eventPath := utils.Substitute(vocab.GetEventsPath, vars)
	//resp, err := cl.Get(eventPath)
	stat := cl.PubAction(thingID, outbox.ReadLatestMethod, nil)
	require.Empty(t, stat.Error)
	require.NotNil(t, stat.Reply)

	// the event must be in the store
	r.Stop()
}
