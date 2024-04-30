package directory_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/thingValues"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/directory/service"
	"github.com/hiveot/hub/runtime/protocols/direct"
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
	svc *service.DirectoryService,
	mt api.IMessageTransport,
	stopFn func()) {

	if clean {
		_ = os.Remove(dirStorePath)
	}
	store := kvbtree.NewKVStore(dirStorePath)
	err := store.Open()
	if err != nil {
		panic("unable to open the digital twin bucket store")
	}

	svc = service.NewDirectoryService(store)
	err = svc.Start()
	if err != nil {
		panic("unable to start the directory service")
	}

	// use direct transport to pass messages to the service
	msgHandler := directory.GetActionHandler(svc)
	mt = direct.NewDirectTransport(thingValues.ThingID, msgHandler)

	return svc, mt, func() {
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
	tdList, err := directory.ReadThings(mt, 0, 10)
	//tdList, err := svc.Directory.ReadThings(0, 10)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	assert.Equal(t, len(thingIDs), len(tdList))

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, mt, stopFunc = startDirectory(false)
	defer stopFunc()
	tdList2, err := directory.ReadThings(mt, 0, 10)
	assert.Equal(t, len(thingIDs), len(tdList2))
}

func TestAddRemoveTD(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := svc.UpdateThing(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	td2json, err := directory.ReadThing(mt, thing1ID)
	td2 := things.TD{}
	err = json.Unmarshal([]byte(td2json), &td2)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

	// after removal, getTD should return nil
	err = directory.RemoveThing(mt, thing1ID)
	assert.NoError(t, err)

	td3, err := directory.ReadThing(mt, thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleTDEvent(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startDirectory(true)
	//msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(thing1ID, title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thing1ID,
		vocab.EventTypeTD, tdDoc1Json, senderID)
	_, err := svc.HandleTDEvent(msg)
	assert.NoError(t, err)

	// non-events like actions should be ignored
	msg.MessageType = vocab.MessageTypeAction
	_, err = svc.HandleTDEvent(msg)
	assert.NoError(t, err)

	tdList, err := directory.ReadThings(mt, 0, 10)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, mt, stopFunc := startDirectory(true)
	_ = svc
	defer stopFunc()
	tds, err := directory.ReadThings(mt, 0, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	tds, err = directory.ReadThings(mt, 10, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	// bad clientID
	tdd1, err := directory.ReadThing(mt, "badid")
	require.Error(t, err)
	require.Empty(t, tdd1)
}

func TestListTDs(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)

	err := svc.UpdateThing(senderID, thing1ID, tdDoc1)
	require.NoError(t, err)

	tdList, err := directory.ReadThings(mt, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, tdList)
	require.True(t, len(tdList) > 0)

	td0 := things.TD{}
	err = json.Unmarshal([]byte(tdList[0]), &td0)
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
