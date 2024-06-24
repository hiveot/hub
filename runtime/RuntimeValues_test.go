package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
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
	cl2, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl2.Disconnect()

	// consumer publish an action to the agent
	stat := cl2.PubAction(dtThing1ID, key1, data)
	require.Empty(t, stat.Error)

	// read the latest actions from the digitwin inbox
	args := digitwin.InboxReadLatestArgs{ThingID: dtThing1ID}
	resp := things.ThingMessageMap{}
	err := cl2.Rpc(digitwin.InboxDThingID, digitwin.InboxReadLatestMethod, &args, &resp)
	require.NoError(t, err)

	actionMsg := resp[key1]
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
	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	// FIXME: this event reaches the agent but it hasn't subscribed. (unnecesary traffic)
	// FIXME: todo subscription is not implemented in embedded and https clients
	// FIXME: todo authorization to publish an event - middleware or transport?
	// FIXME: unmarshal error
	err := hc.PubEvent(agThingID, key1, data)
	assert.NoError(t, err)

	// consumer reads the posted event
	cl, token := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl.Disconnect()
	// read using a plain old http client
	hostPort := fmt.Sprintf("localhost:%d", ts.Port)
	tlsClient := tlsclient.NewTLSClient(hostPort, ts.Certs.CaCert, time.Minute)
	tlsClient.ConnectWithToken(token)

	// read latest using the experimental http REST API
	vars := map[string]string{"thingID": dtThingID}
	eventPath := utils.Substitute(vocab.GetEventsPath, vars)
	reply, err := tlsClient.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, reply)

	d2 := make(map[string]things.ThingMessage)
	err = json.Unmarshal(reply, &d2)

	tmm1 := things.ThingMessageMap{}
	err = json.Unmarshal(reply, &tmm1)
	require.NoError(t, err)
	require.NotZero(t, len(tmm1))

	// read latest using the generated client
	resp, err := digitwin.OutboxReadLatest(hc, nil, "", dtThingID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	tmm2 := things.ThingMessageMap{}
	err = json.Unmarshal([]byte(resp), &tmm2)
	require.NoError(t, err)
	require.Equal(t, len(tmm1), len(tmm2))
}
