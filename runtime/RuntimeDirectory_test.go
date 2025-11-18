package runtime_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/server/discoserver"
	"github.com/hiveot/hivekit/go/utils/tlsclient"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	authn "github.com/hiveot/hub/runtime/authn/api"
	authz "github.com/hiveot/hub/runtime/authz/api"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FIXME: add tests to check that the runtime requests the TDs of newly connected agents

func TestAddRemoveTD(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const agentID = "agent1"
	const userID = "user1"
	const agThing1ID = "thing1"

	r := startRuntime()
	defer r.Stop()

	// Create the agent and its TDs to query
	td1 := td.NewTD(agThing1ID, "Title", vocab.ThingSensorMulti)
	td1JSON, _ := jsoniter.MarshalToString(td1)
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()

	// Create the consumer
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// Add the TD
	err := digitwin.ThingDirectoryUpdateThing(ag1.Consumer, td1JSON)
	assert.NoError(t, err)

	// Get returns a serialized TD object
	// use the helper directory client rpc method
	var dtThing1ID = td.MakeDigiTwinThingID(agentID, agThing1ID)
	td3JSON, err := digitwin.ThingDirectoryRetrieveThing(ag1.Consumer, dtThing1ID)
	require.NoError(t, err)
	var td3 td.TD
	err = jsoniter.UnmarshalFromString(td3JSON, &td3)
	require.NoError(t, err)
	assert.Equal(t, dtThing1ID, td3.ID)

	// RemoveTD from the directory
	err = co1.Rpc(wot.OpInvokeAction, digitwin.ThingDirectoryDThingID,
		digitwin.ThingDirectoryDeleteThingMethod, dtThing1ID, nil)
	require.NoError(t, err)

	// after removal of the TD, getTD should return an error
	t.Log("Reading non-existing TD should fail")
	td4Json, err := digitwin.ThingDirectoryRetrieveThing(co1, dtThing1ID)
	require.Error(t, err)
	require.Empty(t, td4Json)
}

func TestReadTDs(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const agentID = "agent1"
	const userID = "user1"
	//const agThing1ID = "thing1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, agThing1ID)

	r := startRuntime()
	defer r.Stop()
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	co1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 1200)

	//td1JSON, _ := json.Marshal(td1)
	//stat := ag.PubEvent(agThing1ID, vocab.EventNameTD, td1JSON)
	//assert.Empty(t, stat.Error)

	// GetThings returns a serialized TD object
	// 1. Use actions
	args := digitwin.ThingDirectoryRetrieveAllThingsArgs{Limit: 10}
	tdList1 := []string{}
	err := co1.Rpc(wot.OpInvokeAction, digitwin.ThingDirectoryDThingID,
		digitwin.ThingDirectoryRetrieveAllThingsMethod, args, &tdList1)
	require.NoError(t, err)
	require.True(t, len(tdList1) > 0)

	// 2. Try it the easy way using the generated client code
	tdList2, err := digitwin.ThingDirectoryRetrieveAllThings(co1, 333, 02)
	require.NoError(t, err)
	require.True(t, len(tdList2) > 0)

	// the TD must now have forms
	tdi := td.TD{}
	err = json.Unmarshal([]byte(tdList2[20]), &tdi)
	require.NoError(t, err)
	for name, aff := range tdi.Properties {
		_ = name
		require.True(t, len(aff.Forms) > 0)
		form := aff.Forms[0]
		assert.NotEmpty(t, form.GetOperation())
		assert.NotEmpty(t, form.GetHRef())
	}
	for name, aff := range tdi.Actions {
		_ = name
		require.True(t, len(aff.Forms) > 0)
		form := aff.Forms[0]
		assert.NotEmpty(t, form.GetOperation())
		assert.NotEmpty(t, form.GetHRef())
	}
	for name, aff := range tdi.Events {
		_ = name
		require.True(t, len(aff.Forms) > 0)
		form := aff.Forms[0]
		assert.NotEmpty(t, form.GetOperation())
		assert.NotEmpty(t, form.GetHRef())
	}
}

func TestReadTDsRest(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const agentID = "agent1"
	const userID = "user1"

	r := startRuntime()
	defer r.Stop()
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	co1, _, token := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer co1.Disconnect()

	// add a whole bunch of things
	ts.AddTDs(agentID, 100)

	serverURL := ts.GetServerURL(authn.ClientTypeConsumer)
	// FIXME: use the consumer protocol
	urlParts, _ := url.Parse(serverURL)
	cl2 := tlsclient.NewTLSClient(urlParts.Host, nil, ts.Certs.CaCert, time.Second*30)
	cl2.SetAuthToken(token)

	tdJSONList, err := digitwin.ThingDirectoryRetrieveAllThings(co1, 100, 0)
	require.NoError(t, err)

	// tds are sent as an array of JSON, first unpack the array of JSON strings
	tdList, err := td.UnmarshalTDList(tdJSONList)
	require.NoError(t, err)
	require.Equal(t, 100, len(tdList)) // 100 is the given limit

	// check reading a single td
	tdJSON, err := digitwin.ThingDirectoryRetrieveThing(co1, tdList[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, tdJSON)
}

func TestTDEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const agentID = "agent1"
	const userID = "user1"
	tdCount := atomic.Int32{}

	r := startRuntime()
	defer r.Stop()
	ag1, _, _ := ts.AddConnectAgent(agentID)
	defer ag1.Disconnect()
	cl1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	// wait to directory TD updated events
	notifHandler := func(msg *messaging.NotificationMessage) {
		if msg.Operation == vocab.OpSubscribeEvent &&
			msg.ThingID == digitwin.ThingDirectoryDThingID &&
			msg.Name == digitwin.ThingDirectoryEventThingUpdated {

			// decode the TD
			tdi := td.TD{}
			payload := msg.ToString(0)
			err := jsoniter.UnmarshalFromString(payload, &tdi)
			assert.NoError(t, err)
			tdCount.Add(1)
		}
	}
	cl1.SetNotificationHandler(notifHandler)
	err := cl1.Subscribe("", "")
	require.NoError(t, err)
	defer cl1.Unsubscribe("", "")

	time.Sleep(time.Millisecond * 100)
	require.NoError(t, err)

	// add a TD and expect an event from the directory
	ts.AddTDs(agentID, 1)
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, int32(1), tdCount.Load())
}

// Discover the directory on the well-known endpoint
func TestDiscoverDirectory(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const userID = "user1"
	var dirTD *td.TD

	r := startRuntime()
	defer r.Stop()
	cl1, _, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()

	httpSrv := r.TransportsMgr.GetServer(messaging.ProtocolTypeHTTPBasic)
	httpSrv.GetConnectURL()

	tdURL := fmt.Sprintf("%s%s", httpSrv.GetConnectURL(), discoserver.DefaultHttpGetDirectoryTDPath)

	http2Cl := tlsclient.NewHttp2TLSClient(ts.Certs.CaCert, nil, time.Second)
	resp, err := http2Cl.Get(tdURL)
	require.NoError(t, err)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(respBody, &dirTD)
	require.NoError(t, err)

}
