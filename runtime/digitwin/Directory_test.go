package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

var testDirFolder = path.Join(os.TempDir(), "test-directory")
var dirStorePath = path.Join(testDirFolder, "directory.data")

// startService initializes a service and a client
// This doesn't use any transport.
func startDirectory(clean bool) (
	svc *service.DigitwinDirectoryService,
	cl hubclient.IHubClient,
	stopFn func()) {

	if clean {
		_ = os.Remove(dirStorePath)
	}
	store := kvbtree.NewKVStore(dirStorePath)
	err := store.Open()
	if err != nil {
		panic("unable to open the digital twin bucket store")
	}

	svc = service.NewDigitwinDirectory(store, nil)
	err = svc.Start()
	if err != nil {
		panic("unable to start the directory service")
	}

	// use direct transport to pass messages to the service
	msgHandler := digitwin.NewDirectoryHandler(svc)
	cl = embedded.NewEmbeddedClient(digitwin.DirectoryAgentID, msgHandler)

	return svc, cl, func() {
		svc.Stop()
		_ = store.Close()
	}
}

// generate a JSON serialized TD document
func createTDDoc(thingID string, title string) *tdd.TD {
	td := tdd.NewTD(thingID, title, vocab.ThingDevice)
	return td
}

func TestStartStopDirectory(t *testing.T) {
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, hc, stopFunc := startDirectory(true)

	// add TDs
	for _, thingID := range thingIDs {
		td := createTDDoc(thingID, thingID)
		tddjson, _ := json.Marshal(td)
		err := svc.UpdateTD("test", string(tddjson))
		require.NoError(t, err)
	}
	// viewers should be able to read the directory
	tdList, err := digitwin.DirectoryReadTDs(hc, 10, 0)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	assert.Equal(t, len(thingIDs), len(tdList))

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, hc, stopFunc = startDirectory(false)
	defer stopFunc()
	tdList, err = digitwin.DirectoryReadTDs(hc, 10, 0)
	assert.Equal(t, len(thingIDs), len(tdList))
}

func TestAddRemoveTD(t *testing.T) {
	const agentID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"
	var dThing1ID = tdd.MakeDigiTwinThingID(agentID, thing1ID)

	svc, hc, stopFunc := startDirectory(true)
	defer stopFunc()

	// use the native thingID for this TD doc as the directory converts it to
	// the digital twin ID using the given agent that owns the TD.
	tdDoc1 := createTDDoc(thing1ID, title1)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.UpdateTD(agentID, string(tddjson))
	assert.NoError(t, err)

	// use the client wrapper to read
	tdJSON, err := digitwin.DirectoryReadTD(hc, dThing1ID)
	require.NoError(t, err)
	var td tdd.TD
	err = json.Unmarshal([]byte(tdJSON), &td)
	require.NoError(t, err)
	assert.Equal(t, dThing1ID, td.ID)

	// after removal, getTD should return nil
	err = svc.RemoveTD("senderID", dThing1ID)
	assert.NoError(t, err)

	td3, err := digitwin.DirectoryReadTD(hc, dThing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleTDEvent(t *testing.T) {
	const agentID = "agent1"
	const rawThing1ID = "thing1"
	const title1 = "title1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, rawThing1ID)

	svc, hc, stopFunc := startDirectory(true)
	//msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(rawThing1ID, title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	msg := hubclient.NewThingMessage(vocab.MessageTypeEvent, rawThing1ID,
		vocab.EventNameTD, string(tdDoc1Json), agentID)
	stat := svc.HandleTDEvent(msg)
	assert.Empty(t, stat.Error)

	// non-events like actions should be ignored
	msg.MessageType = vocab.MessageTypeAction
	stat = svc.HandleTDEvent(msg)
	assert.Empty(t, stat.Error)

	tdList, err := digitwin.DirectoryReadTDs(hc, 10, 0)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, hc, stopFunc := startDirectory(true)
	_ = svc
	defer stopFunc()
	tdJsonList, err := digitwin.DirectoryReadTDs(hc, 10, 0)
	require.NoError(t, err)
	require.Empty(t, tdJsonList)

	tdJsonList, err = digitwin.DirectoryReadTDs(hc, 10, 10)
	require.NoError(t, err)
	require.Empty(t, tdJsonList)

	// bad clientID
	tdJson, err := digitwin.DirectoryReadTD(hc, "badid")
	require.Error(t, err)
	require.Empty(t, tdJson)
}

//func TestQueryTDs(t *testing.T) {
//	_ = os.Remove(testStoreFile)
//	const senderID = "agent1"
//	const thing1ID = "agent1:thing1"
//	const title1 = "title1"
//
//	svc, stopFunc := startDirectory()
//	defer stopFunc()
//
//	tdDoc1 := createTDDoc(thing1ID, title1)
//	err := svc.UpdateTD(senderID, thing1ID, tdDoc1)
//	require.NoError(t, err)
//
//	jsonPathQuery := `$[?(@.id=="agent1:thing1")]`
//	tdList, err := svc.QueryTDs(jsonPathQuery)
//	require.NoError(t, err)
//	assert.NotNil(t, tdList)
//	assert.True(t, len(tdList) > 0)
//	el0 := things.TD{}
//	json.Decode([]byte(tdList[0]), &el0)
//	assert.Equal(t, thing1ID, el0.ID)
//	assert.Equal(t, title1, el0.Title)
//}
