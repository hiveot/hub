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

func TestHttpsPubValueEvent(t *testing.T) {
	const thingID = "thing1"
	const key1 = "key1"
	const senderID = "sender1"
	const data = "Hello world"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)

	// publish an event
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key1, []byte(data), senderID)

	// TODO: use a client from the library. path needs to match the server
	//evPath := fmt.Sprintf("/event/%s/%s", thingID, key1)
	vars := map[string]string{"thingID": thingID, "key": key1}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)

	_, err := cl.Post(eventPath, msg.Data)
	assert.NoError(t, err)

	props, err := r.ValueSvc.ReadEvents(thingID, nil, "")
	require.NoError(t, err)
	require.NotEmpty(t, props)
	assert.Equal(t, msg.Data, props[key1].Data)

	// the event must be in the store
	r.Stop()
}

func TestHttpsPutProperties(t *testing.T) {
	const thingID = "thing1"
	const agentID = "agent1"

	r := startRuntime()
	cl, token := addConnectClient(r, api.ClientTypeAgent, agentID)
	require.NotEmpty(t, token)
	vars := map[string]string{"thingID": thingID, "key": vocab.EventTypeProperties}
	pubEventPath := utils.Substitute(vocab.PostEventPath, vars)
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}
	data, _ := json.Marshal(props)
	_, err := cl.Post(pubEventPath, data)
	assert.NoError(t, err)

	readPropsPath := utils.Substitute(vocab.GetPropertiesPath, vars)
	data, err = cl.Get(readPropsPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}
