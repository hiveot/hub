package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
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
	cl1, _ := ts.AddConnectAgent(agentID)
	defer cl1.Disconnect()
	// consumer sends actions and receives events
	cl2, _ := ts.AddConnectUser(userID, api.ClientRoleManager)
	defer cl2.Disconnect()

	// consumer publish an action to the agent
	stat := cl2.PubAction(dtThing1ID, key1, []byte(data))
	require.Empty(t, stat.Error)

	// read the latest actions from the digitwin inbox
	args := digitwin.InboxReadLatestArgs{ThingID: dtThing1ID}
	resp := digitwin.InboxReadLatestResp{}
	err := cl2.Rpc(digitwin.InboxDThingID, digitwin.InboxReadLatestMethod, &args, &resp)
	require.NoError(t, err)

	actionMsg := resp.ThingValues[key1]
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
	cl1, _ := ts.AddConnectAgent(agentID)
	defer cl1.Disconnect()

	// FIXME: this event reaches the agent but it hasn't subscribed. (unnecesary traffic)
	// FIXME: todo subscription is not implemented in embedded and https clients
	// FIXME: todo authorization to publish an event - middleware or transport?
	// FIXME: unmarshal error
	err := cl1.PubEvent(agThingID, key1, []byte(data))
	assert.NoError(t, err)

	// consumer reads the posted event
	cl, token := ts.AddConnectUser(userID, api.ClientRoleManager)
	defer cl.Disconnect()
	// read using a plain old http client
	hostPort := fmt.Sprintf("localhost:%d", ts.Port)
	tlsClient := tlsclient.NewTLSClient(hostPort, ts.Certs.CaCert, time.Minute)
	tlsClient.ConnectWithToken(token)

	// read latest using the experimental http REST API (which needs a swagger definition)
	vars := map[string]string{"thingID": dtThingID}
	eventPath := utils.Substitute(vocab.GetEventsPath, vars)
	reply, err := tlsClient.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, reply)
	tmm := things.ThingMessageMap{}
	err = json.Unmarshal(reply, &tmm)
	require.NoError(t, err)
	require.NotZero(t, len(tmm))

	// read latest using the generated RPC client
	args := digitwin.OutboxReadLatestArgs{ThingID: dtThingID}
	outboxCl := digitwin.NewOutboxClient(cl)
	resp, err := outboxCl.ReadLatest(args)
	require.NoError(t, err)
	require.NotNil(t, resp)
	tmm = things.ThingMessageMap{}
	err = json.Unmarshal([]byte(resp.Values), &tmm)
}
