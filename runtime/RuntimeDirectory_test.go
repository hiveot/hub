package runtime_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
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
	const thing1ID = "thing1"

	r := startRuntime()
	defer r.Stop()
	ag, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer ag.Close()
	cl, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl.Close()

	td1 := createTD(thing1ID)
	params := map[string]string{"thingID": thing1ID}
	addr := utils.Substitute(vocab.PostThingPath, params)
	_, err := ag.Post(addr, td1)
	assert.NoError(t, err)

	// Get returns a serialized TD object
	addr = utils.Substitute(vocab.GetThingPath, params)
	td2Doc, err := cl.Get(addr)
	//time.Sleep(time.Second * 30)  // otherwise timeout during debugging. Is there a better way?
	require.NoError(t, err)
	td2 := things.TD{}
	err = json.Unmarshal(td2Doc, &td2)
	require.NoError(t, err)

	addr = fmt.Sprintf("/things/%s", thing1ID)
	_, err = cl.Delete(addr)
	require.NoError(t, err)

	// after removal, getTD should return nil
	addr = fmt.Sprintf("/things/%s", thing1ID)
	td3, err := cl.Get(addr)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestReadThings(t *testing.T) {

	const agentID = "agent1"
	const userID = "user1"
	const thing1ID = "thing1"

	r := startRuntime()
	defer r.Stop()
	ag, _ := addConnectClient(r, api.ClientTypeAgent, agentID)
	defer ag.Close()
	cl, _ := addConnectClient(r, api.ClientTypeUser, userID)
	defer cl.Close()

	td1 := createTD(thing1ID)
	params := map[string]string{"thingID": thing1ID}
	addr := utils.Substitute(vocab.PostThingPath, params)
	_, err := ag.Post(addr, td1)
	assert.NoError(t, err)

	// GetThings returns a serialized TD object
	addr = utils.Substitute(vocab.GetThingsPath, params)
	resp, err := cl.Get(addr)
	require.NoError(t, err)
	jsonList := []string{}
	err = json.Unmarshal(resp, &jsonList)
	require.NoError(t, err)

	td2 := things.TD{}
	err = json.Unmarshal([]byte(jsonList[0]), &td2)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

}
