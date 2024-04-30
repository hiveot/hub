package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHttpsGetActions(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const agentID = "agent1"
	const userID = "user1"
	const data = "Hello world"

	r := startRuntime()

	cl1, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	vars := map[string]string{"thingID": thingID, "key": key1}
	actionPath := utils.Substitute(vocab.PostActionPath, vars)
	_, err := cl1.Post(actionPath, []byte(data))
	require.NoError(t, err)

	cl2, _ := addConnectClient(r, api.ClientTypeUser, userID)
	vars = map[string]string{"thingID": thingID, "key": key1}
	eventPath := utils.Substitute(vocab.GetActionsPath, vars)
	resp, err := cl2.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, resp)
	msg := things.ThingMessageMap{}
	err = json.Unmarshal(resp, &msg)
	require.NoError(t, err)
	assert.Equal(t, data, string(msg[key1].Data))

	// the event must be in the store
	r.Stop()
}

func TestHttpsGetEvents(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const data = "Hello world"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)

	vars := map[string]string{"thingID": thingID}
	eventPath := utils.Substitute(vocab.GetEventsPath, vars)

	resp, err := cl.Get(eventPath)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// the event must be in the store
	r.Stop()
}
