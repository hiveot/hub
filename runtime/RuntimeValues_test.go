package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient/httpsse"
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
	args := digitwin.InboxReadLatestArgs{ThingID: dtThing1ID, Key: key1}
	resp := digitwin.InboxRecord{}
	err := cl2.Rpc(digitwin.InboxDThingID, digitwin.InboxReadLatestMethod, &args, &resp)
	require.NoError(t, err)
	require.Equal(t, stat.MessageID, resp.MessageID)
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

	err := hc.PubEvent(agThingID, key1, data)
	assert.NoError(t, err)

	// consumer reads the posted event
	cl, token := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl.Disconnect()
	// read using a plain old http client
	hostPort := fmt.Sprintf("localhost:%d", ts.Port)
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, ts.Certs.CaCert, time.Minute)
	tlsClient.SetAuthToken(token)

	// read latest using the experimental http REST API
	vars := map[string]string{"thingID": dtThingID}
	eventPath := utils.Substitute(httpsse.GetReadAllEventsPath, vars)
	reply, _, err := tlsClient.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, reply)

	tmm1 := things.ThingMessageMap{}
	err = json.Unmarshal(reply, &tmm1)
	require.NoError(t, err)
	require.NotZero(t, len(tmm1))

	// read latest using the generated client
	resp, err := digitwin.OutboxReadLatest(hc, nil, "", "", dtThingID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	tmm2, err := things.NewThingMessageMapFromSource(resp)
	require.NoError(t, err)
	require.Equal(t, len(tmm1), len(tmm2))
}

// Get events from the outbox using the experimental http REST api
func TestHttpsGetProps(t *testing.T) {
	const agentID = "agent1"
	const agThingID = "thing1"
	const key1 = "key1"
	const key2 = "key2"
	const userID = "user1"
	const data1 = "Hello world"
	const data2 = 25
	var dtThingID = things.MakeDigiTwinThingID(agentID, agThingID)

	r := startRuntime()
	defer r.Stop()
	// agent publishes events
	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	propMap := map[string]any{}
	propMap[key1] = data1
	propMap[key2] = data2
	err := hc.PubProps(agThingID, propMap)
	require.NoError(t, err)
	//
	// consumer read properties
	cl2, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl2.Disconnect()
	data, err := digitwin.OutboxReadLatest(hc, nil, "", "", dtThingID)
	require.NoError(t, err)

	vmm, err := things.NewThingMessageMapFromSource(data)
	require.NoError(t, err)
	// note: golang unmarshals integers as float64.
	data2raw := vmm[key2].Data.(float64)
	require.Equal(t, data2, int(data2raw))
}
