package runtime_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
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
	ag, _ := ts.AddConnectAgent(agentID)
	ag.SetActionHandler(func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		stat.Status = hubclient.DeliveryCompleted
		return
	})
	defer ag.Disconnect()
	cl, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	cl.SetEventHandler(func(msg *things.ThingMessage) error {
		return nil
	})
	defer cl.Disconnect()

	// Add the TD by sending it as an event
	td1 := things.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := json.Marshal(td1)
	err := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	assert.NoError(t, err)

	// Get returns a serialized TD object
	// use the helper directory client rpc method
	td3Json, err := digitwin.DirectoryReadTD(cl, dtThing1ID)
	require.NoError(t, err)
	var td3 things.TD
	err = json.Unmarshal([]byte(td3Json), &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	//stat = cl.Rpc(nil, directory.ThingID, directory.RemoveTDMethod, &args, nil)
	args4JSON, _ := json.Marshal(dtThing1ID)
	stat := cl.PubAction(digitwin.DirectoryDThingID, digitwin.DirectoryRemoveTDMethod, args4JSON)
	require.Empty(t, stat.Error)

	// after removal of the TD, getTD should return an error but delivery is successful
	stat = cl.PubAction(digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, args4JSON)
	require.NotEmpty(t, stat.Error)
	require.Equal(t, hubclient.DeliveryCompleted, stat.Status)
}

func TestReadTDs(t *testing.T) {
	t.Log("--- TestReadTDs start ---")

	const agentID = "agent1"
	const userID = "user1"
	//const agThing1ID = "thing1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()
	cl, _ := ts.AddConnectUser(userID, authn.ClientRoleManager)
	defer cl.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 1200)

	//td1JSON, _ := json.Marshal(td1)
	//stat := ag.PubEvent(agThing1ID, vocab.EventTypeTD, td1JSON)
	//assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// try this with the generated client api and the client rpc method
	tdList, err := digitwin.DirectoryReadTDs(cl, 333, 02)
	require.NoError(t, err)
	require.Greater(t, len(tdList), 3)

}
