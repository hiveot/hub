package directory_test

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/directory"
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
func createTDDoc(thingID string, title string) string {
	td := &things.TD{
		ID:    thingID,
		Title: title,
	}
	tdDoc, _ := json.Marshal(td)
	return string(tdDoc)
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
	tdList, err := svc.GetTDs(0, 10)
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
	err := svc.UpdateTD(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	tdj2, err := svc.GetTD(thing1ID)
	var td2 things.TD
	err = json.Unmarshal([]byte(tdj2), &td2)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)
	assert.Equal(t, tdDoc1, tdj2)

	// after removal, getTD should return nil
	err = svc.RemoveTD(senderID, thing1ID)
	assert.NoError(t, err)

	td3, err := svc.GetTD(thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleEvent(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(thing1ID, title1)
	tv := things.NewThingMessage(vocab.MessageTypeEvent, thing1ID,
		vocab.EventTypeTD, []byte(tdDoc1), senderID)
	_, err := svc.HandleEvent(tv)
	assert.NoError(t, err)

	// non-events like actions should be ignored
	tv.MessageType = vocab.MessageTypeAction
	_, err = svc.HandleEvent(tv)
	assert.NoError(t, err)

	tdList, err := svc.GetTDs(0, 10)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, stopFunc := startDirectory(true)
	defer stopFunc()
	tds, err := svc.GetTDs(0, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	tds, err = svc.GetTDs(10, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	// missing data
	cursorKey, err := svc.CursorMgr().NewCursor(clientID)
	require.NoError(t, err)
	_, _, valid, err := svc.CursorMgr().First(cursorKey, clientID)
	require.NoError(t, err)
	require.False(t, valid)
	_, _, valid, err = svc.CursorMgr().Next(cursorKey, clientID)
	require.NoError(t, err)
	require.False(t, valid)
	_, valid, err = svc.CursorMgr().NextN(cursorKey, clientID, 1)
	require.NoError(t, err)
	require.False(t, valid)

	// bad clientID
	_, _, valid, err = svc.CursorMgr().First(cursorKey, "badid")
	require.Error(t, err)
	require.False(t, valid)

	// bad cursorKey
	_, _, valid, err = svc.CursorMgr().First("badkey", clientID)
	require.Error(t, err)
	require.False(t, valid)
	_, _, valid, err = svc.CursorMgr().Next("badkey", clientID)
	require.Error(t, err)
	require.False(t, valid)
	_, valid, err = svc.CursorMgr().NextN("badkey", clientID, 1)
	require.Error(t, err)
	require.False(t, valid)

	svc.CursorMgr().CloseCursor(cursorKey, clientID)
}

func TestListTDs(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)

	err := svc.UpdateTD(senderID, thing1ID, tdDoc1)
	require.NoError(t, err)

	tdList, err := svc.GetTDs(0, 10)
	require.NoError(t, err)
	assert.NotNil(t, tdList)
	assert.True(t, len(tdList) > 0)
	//	slog.Infof("--- TestListTDs end ---")
}

func TestCursor(t *testing.T) {
	const clientID = "client1"
	const publisherID = "urn:test"
	const thing1ID = "urn:agent1:thing1"
	const thing2ID = "urn:agent1:thing2"
	const thing3ID = "urn:agent1:thing3"

	svc, stopFunc := startDirectory(true)
	defer stopFunc()

	// add 3 docs.
	tdDoc1 := createTDDoc(thing1ID, "title 1")
	err := svc.UpdateTD(publisherID, thing1ID, tdDoc1)
	require.NoError(t, err)
	tdDoc2 := createTDDoc(thing2ID, "title 2")
	err = svc.UpdateTD(publisherID, thing2ID, tdDoc2)
	require.NoError(t, err)
	tdDoc3 := createTDDoc(thing3ID, "title 3")
	err = svc.UpdateTD(publisherID, thing3ID, tdDoc3)
	require.NoError(t, err)

	// expect 3 docs, two service capabilities and the one just added
	cursorKey, err := svc.CursorMgr().NewCursor(clientID)
	require.NoError(t, err)
	defer svc.CursorMgr().CloseCursor(cursorKey, clientID)

	thingID, tdValue, valid, err := svc.CursorMgr().First(cursorKey, clientID)
	require.NotEmpty(t, thingID)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, tdValue)

	_, tdValue, valid, err = svc.CursorMgr().Next(cursorKey, clientID) // second
	assert.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, tdValue)

	tdds, remaining, err := svc.CursorMgr().NextN(cursorKey, clientID, 3) // there is no 4th
	assert.NoError(t, err)
	assert.False(t, remaining)
	assert.True(t, len(tdds) == 1)

	tdValues, valid, err := svc.CursorMgr().NextN(cursorKey, clientID, 10) // still no third
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, tdValues)
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
