package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddRemoveTD(t *testing.T) {
	t.Log("--- TestAddRemoveTD start ---")
	defer t.Log("--- TestAddRemoveTD end ---")

	const agentID = "agent1"
	const userID = "user1"
	const agThing1ID = "thing1"
	var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	ag.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		stat.Status = api.DeliveryCompleted
		return
	})
	defer ag.Disconnect()
	cl, _ := addConnectClient(r, api.ClientTypeUser, userID)
	cl.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		stat.Status = api.DeliveryCompleted
		return
	})
	defer cl.Disconnect()

	td1 := createTD(agThing1ID)
	td1JSON, _ := json.Marshal(td1)
	stat := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	assert.Equal(t, api.DeliveryCompleted, stat.Status)
	assert.Empty(t, stat.Error)

	// Get returns a serialized TD object
	args := directory.ReadThingArgs{ThingID: dtThing1ID}
	argsJSON, _ := json.Marshal(args)
	stat = cl.PubAction(directory.ThingID, directory.ReadThingMethod, argsJSON)
	require.Empty(t, stat.Error) // no client handler error
	require.Equal(t, api.DeliveryCompleted, stat.Status)
	td2 := things.TD{}
	err := json.Unmarshal(stat.Reply, &td2)
	require.NoError(t, err)

	args2 := directory.RemoveThingArgs{ThingID: dtThing1ID}
	args2JSON, _ := json.Marshal(args2)
	stat = cl.PubAction(directory.ThingID, directory.RemoveThingMethod, args2JSON)
	require.Empty(t, stat.Error)

	// TODO also use the client rpc method
	//stat = cl.Rpc(nil, directory.ThingID, directory.RemoveThingMethod, &args,nil)

	// after removal, getTD should return an error but delivery is successful
	stat = cl.PubAction(directory.ThingID, directory.ReadThingMethod, args2JSON)
	require.NotEmpty(t, stat.Error)
	require.Equal(t, api.DeliveryCompleted, stat.Status)
}

func TestReadThings(t *testing.T) {
	t.Log("--- TestReadThings start ---")

	const agentID = "agent1"
	const userID = "user1"
	const agThing1ID = "thing1"
	var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer ag.Disconnect()
	cl, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl.Disconnect()

	td1 := createTD(agThing1ID)
	td1JSON, _ := json.Marshal(td1)
	stat := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// try this with the generated client api and the client rpc method
	args := directory.ReadThingsArgs{Limit: 10}
	resp, stat, err := directory.ReadThings(cl.Rpc, args)
	require.NoError(t, err)

	td2 := make(map[string]interface{})
	err = json.Unmarshal([]byte(resp.Output[0]), &td2)
	require.NoError(t, err)

	td3 := things.TD{}
	err = json.Unmarshal([]byte(resp.Output[0]), &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

}
