package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/things"
	service2 "github.com/hiveot/hub/runtime/digitwin/service"
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
	svc *service2.DigitwinDirectory,
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

	svc = service2.NewDigitwinDirectory(store)
	err = svc.Start()
	if err != nil {
		panic("unable to start the directory service")
	}

	// use direct transport to pass messages to the service
	msgHandler := directory.NewActionHandler(svc)
	cl = embedded.NewEmbeddedClient(directory.ThingID, msgHandler)
	//mt = direct.NewDirectTransport(directory.ThingID, msgHandler)

	return svc, cl, func() {
		svc.Stop()
		_ = store.Close()
	}
}

// generate a JSON serialized TD document
func createTDDoc(thingID string, title string) *things.TD {
	td := things.NewTD(thingID, title, vocab.ThingDevice)
	return td
}

func TestStartStopDirectory(t *testing.T) {
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, mt, stopFunc := startDirectory(true)

	// add TDs
	for _, thingID := range thingIDs {
		td := createTDDoc(thingID, thingID)
		err := svc.UpdateThing("test", thingID, td)
		require.NoError(t, err)
	}
	// viewers should be able to read the directory
	args := directory.ReadTDsArgs{Limit: 10, Offset: 0}
	resp, err := directory.ReadTDs(mt, args)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	assert.Equal(t, len(thingIDs), len(resp.Output))

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, mt, stopFunc = startDirectory(false)
	defer stopFunc()
	args = directory.ReadTDsArgs{Limit: 10, Offset: 0}
	resp, err = directory.ReadTDs(mt, args)
	assert.Equal(t, len(thingIDs), len(resp.Output))
}

func TestAddRemoveTD(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, cl, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := svc.UpdateThing(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	resp, err := directory.ReadTDs(cl, directory.ReadTDsArgs{Limit: 10})
	td2 := things.TD{}
	err = json.Unmarshal([]byte(resp.Output[0]), &td2)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

	// after removal, getTD should return nil
	err = directory.RemoveTD(cl, directory.RemoveTDArgs{ThingID: thing1ID})
	assert.NoError(t, err)

	td3, err := directory.ReadTD(cl, directory.ReadTDArgs{ThingID: thing1ID})
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleTDEvent(t *testing.T) {
	const agentID = "agent1"
	const rawThing1ID = "thing1"
	const title1 = "title1"
	//var dtThing1ID = things.MakeDigiTwinThingID(agentID, rawThing1ID)

	svc, cl, stopFunc := startDirectory(true)
	//msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(rawThing1ID, title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	msg := things.NewThingMessage(vocab.MessageTypeEvent, rawThing1ID,
		vocab.EventTypeTD, tdDoc1Json, agentID)
	stat := svc.HandleTDEvent(msg)
	assert.Empty(t, stat.Error)

	// non-events like actions should be ignored
	msg.MessageType = vocab.MessageTypeAction
	stat = svc.HandleTDEvent(msg)
	assert.Empty(t, stat.Error)

	resp, err := directory.ReadTDs(cl, directory.ReadTDsArgs{Limit: 10})
	assert.Equal(t, 1, len(resp.Output))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, cl, stopFunc := startDirectory(true)
	_ = svc
	defer stopFunc()
	resp, err := directory.ReadTDs(cl, directory.ReadTDsArgs{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, resp.Output)

	resp, err = directory.ReadTDs(cl, directory.ReadTDsArgs{Limit: 10, Offset: 10})
	require.NoError(t, err)
	require.Empty(t, resp.Output)

	// bad clientID
	resp2, err := directory.ReadTD(cl, directory.ReadTDArgs{ThingID: "badid"})
	require.Error(t, err)
	require.Empty(t, resp2.Output)
}

func TestListTDs(t *testing.T) {
	const agentID = "agent1"
	const rawThing1ID = "thing1"
	const title1 = "title1"
	var dtThing1ID = things.MakeDigiTwinThingID(agentID, rawThing1ID)

	svc, cl, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(dtThing1ID, title1)

	err := svc.UpdateThing(agentID, dtThing1ID, tdDoc1)
	require.NoError(t, err)

	resp, err := directory.ReadTDs(cl, directory.ReadTDsArgs{Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, resp.Output)
	require.True(t, len(resp.Output) > 0)

	td0 := things.TD{}
	err = json.Unmarshal([]byte(resp.Output[0]), &td0)
	require.NoError(t, err)
	//	slog.Infof("--- TestListTDs end ---")
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
//	json.Unmarshal([]byte(tdList[0]), &el0)
//	assert.Equal(t, thing1ID, el0.ID)
//	assert.Equal(t, title1, el0.Title)
//}
