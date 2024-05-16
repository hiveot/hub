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
	ag, _ := ts.AddConnectClient(api.ClientTypeAgent, agentID, api.ClientRoleAgent)
	ag.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		stat.Status = api.DeliveryCompleted
		return
	})
	defer ag.Disconnect()
	cl, _ := ts.AddConnectClient(api.ClientTypeUser, userID, api.ClientRoleManager)
	cl.SetMessageHandler(func(msg *things.ThingMessage) (stat api.DeliveryStatus) {
		stat.Status = api.DeliveryCompleted
		return
	})
	defer cl.Disconnect()

	// Add the TD by sending it as an event
	td1 := things.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := json.Marshal(td1)
	stat := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	assert.Equal(t, api.DeliveryCompleted, stat.Status)
	assert.Empty(t, stat.Error)

	// Get returns a serialized TD object
	args := directory.ReadTDArgs{ThingID: dtThing1ID}
	argsJSON, _ := json.Marshal(args)
	stat, err := cl.PubAction(directory.ThingID, directory.ReadTDMethod, argsJSON)
	require.NoError(t, err) // no client handler error
	require.Equal(t, api.DeliveryCompleted, stat.Status)

	// double unmarshal needed, one for message, second for TD payload
	resp := directory.ReadTDResp{}
	err = json.Unmarshal(stat.Reply, &resp)
	require.NoError(t, err)
	td2 := things.TD{}
	err = json.Unmarshal([]byte(resp.Output), &td2)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td2.ID)

	// use the generated directory client rpc method
	td3 := things.TD{}
	resp, err = directory.ReadTD(cl, args)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(resp.Output), &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	//stat = cl.Rpc(nil, directory.ThingID, directory.RemoveTDMethod, &args, nil)
	args4 := directory.RemoveTDArgs{ThingID: dtThing1ID}
	args4JSON, _ := json.Marshal(args4)
	stat, err = cl.PubAction(directory.ThingID, directory.RemoveTDMethod, args4JSON)
	require.Empty(t, stat.Error)

	// after removal, getTD should return an error but delivery is successful
	stat, err = cl.PubAction(directory.ThingID, directory.ReadTDMethod, args4JSON)
	require.Error(t, err)
	require.Equal(t, api.DeliveryCompleted, stat.Status)
}

func TestReadTDs(t *testing.T) {
	t.Log("--- TestReadTDs start ---")

	const agentID = "agent1"
	const userID = "user1"
	//const agThing1ID = "thing1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectClient(api.ClientTypeAgent, agentID, api.ClientRoleAgent)
	defer ag.Disconnect()
	cl, _ := ts.AddConnectClient(api.ClientTypeUser, userID, api.ClientRoleManager)
	defer cl.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 1200)

	//td1JSON, _ := json.Marshal(td1)
	//stat := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	//assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// try this with the generated client api and the client rpc method
	args := directory.ReadTDsArgs{Offset: 02, Limit: 333}
	resp, err := directory.ReadTDs(cl, args)
	require.NoError(t, err)

	td2 := make(map[string]interface{})
	err = json.Unmarshal([]byte(resp.Output[0]), &td2)
	require.NoError(t, err)

	td3 := things.TD{}
	err = json.Unmarshal([]byte(resp.Output[0]), &td3)
	require.NoError(t, err)

}
