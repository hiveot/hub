package directory_test

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/directory"
	"github.com/hiveot/hub/runtime/directory/rpc"
	"github.com/hiveot/hub/runtime/directory/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

var testFolder = path.Join(os.TempDir(), "test-directory")
var testStoreFile = path.Join(testFolder, "directory.json")

// startDirectory initializes a Directory service
func startDirectory(clean bool) (svc *service.DirectoryService, stopFn func()) {
	if clean {
		_ = os.Remove(testStoreFile)
	}
	store := kvbtree.NewKVStore(testStoreFile)
	err := store.Open()
	if err != nil {
		panic("unable to open directory store")
	}
	cfg := directory.NewDirectoryConfig()
	svc = service.NewDirectoryService(&cfg, store)
	err = svc.Start()
	if err != nil {
		panic("unable to start directory")
	}

	return svc, func() {
		svc.Stop()
		_ = store.Close()
	}
}

// generate a JSON serialized TD document
func createTDDoc(thingID string, title string) *things.TD {
	td := things.NewTD(thingID, title, vocab.ThingDevice)
	return td
}

func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	err = os.MkdirAll(testFolder, 0700)

	if err != nil {
		panic(err)
	}

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	// viewers should be able to read the directory
	tdList, err := svc.ReadTDDs(0, 10)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	_ = tdList
}

func TestAddRemoveTD(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := svc.UpdateTDD(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	td2, err := svc.ReadTDD(thing1ID)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

	// after removal, getTD should return nil
	err = svc.RemoveTDD(senderID, thing1ID)
	assert.NoError(t, err)

	td3, err := svc.ReadTDD(thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleEvent(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, stopFunc := startDirectory(true)
	dirRPC := rpc.NewDirectoryRPC(svc)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(thing1ID, title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	tv := things.NewThingMessage(vocab.MessageTypeEvent, thing1ID,
		vocab.EventTypeTD, tdDoc1Json, senderID)
	_, err := dirRPC.HandleMessage(tv)
	assert.NoError(t, err)

	// non-events like actions should be ignored
	tv.MessageType = vocab.MessageTypeAction
	_, err = dirRPC.HandleMessage(tv)
	assert.NoError(t, err)

	tdList, err := svc.ReadTDDs(0, 10)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, stopFunc := startDirectory(true)
	defer stopFunc()
	tds, err := svc.ReadTDDs(0, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	tds, err = svc.ReadTDDs(10, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	// bad clientID
	tdd1, err := svc.ReadTDD("badid")
	require.Error(t, err)
	require.Nil(t, tdd1)
}

func TestListTDs(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)

	err := svc.UpdateTDD(senderID, thing1ID, tdDoc1)
	require.NoError(t, err)

	tdList, err := svc.ReadTDDs(0, 10)
	require.NoError(t, err)
	assert.NotNil(t, tdList)
	assert.True(t, len(tdList) > 0)
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
