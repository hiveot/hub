package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHttpsPubValueEvent(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const data = "Hello world"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)

	// publish an event
	//msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key1, []byte(data), senderID)

	// TODO: use a client from the library. path needs to match the server
	//evPath := fmt.Sprintf("/event/%s/%s", thingID, key1)
	//vars := map[string]string{"thingID": thingID, "key": key1}
	//eventPath := utils.Substitute(vocab.PostEventPath, vars)

	stat := cl.PubEvent(thingID, key1, []byte(data))
	assert.Empty(t, stat.Error)

	// Read events
	args := outbox.ReadLatestArgs{ThingID: thingID}
	resp, stat, err := outbox.ReadLatest(cl.Rpc, args)
	require.NoError(t, err)
	_ = resp

	reply := things.ThingMessageMap{}
	err = json.Unmarshal(stat.Reply, &reply)

	//props, err := outbox.ReadLatest(cl, thingID, nil, "")
	//props, err := r.DigitwinSvc.Outbox.ReadLatest(thingID, nil, "")
	require.NoError(t, err)
	require.NotEmpty(t, reply)
	assert.Equal(t, []byte(data), reply[key1].Data)

	// the event must be in the store
	r.Stop()
}

func TestHttpsPutProperties(t *testing.T) {
	const thingID = "thing1"
	const agentID = "agent1"

	r := startRuntime()
	cl, token := addConnectClient(r, api.ClientTypeAgent, agentID)
	require.NotEmpty(t, token)
	//vars := map[string]string{"thingID": thingID, "key": vocab.EventTypeProperties}
	//pubEventPath := utils.Substitute(vocab.PostEventPath, vars)
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}
	data, _ := json.Marshal(props)
	stat := cl.PubEvent(thingID, vocab.EventTypeProperties, data)
	assert.Empty(t, stat.Error)

	//readPropsPath := utils.Substitute(vocab.GetPropertiesPath, vars)
	//data, err = cl.Get(readPropsPath)
	args := outbox.ReadLatestArgs{ThingID: thingID}
	resp, stat, err := outbox.ReadLatest(cl.Rpc, args)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Values)
}
