package directory_test

import (
	"encoding/json"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/core/directory/service"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hub/lib/logging"
)

var testFolder = path.Join(os.TempDir(), "test-directory")
var testStoreFile = path.Join(testFolder, "directory.json")
var core = "mqtt"

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

const serviceID string = directoryapi.ServiceName

// startDirectory initializes a Directory service, optionally using capnp RPC
func startDirectory() (
	r *dirclient.ReadDirectoryClient, u *dirclient.UpdateDirectoryClient, stopFn func()) {

	slog.Info("startDirectory start")
	defer slog.Info("startDirectory ended")
	_ = os.Remove(testStoreFile)
	store := kvbtree.NewKVStore(serviceID, testStoreFile)
	err := store.Open()
	if err != nil {
		panic("unable to open directory store")
	}

	// the service needs a server connection
	hc1, err := testServer.AddConnectClient(
		serviceID, authapi.ClientTypeService, authapi.ClientRoleService)
	svc := service.NewDirectoryService(hc1, store)
	if err == nil {
		err = svc.Start()
	}
	if err != nil {
		panic("service fails to start: " + err.Error())
	}

	// connect as a user to the server above
	hc2, err := testServer.AddConnectClient(
		"admin", authapi.ClientTypeUser, authapi.ClientRoleAdmin)
	readCl := dirclient.NewReadDirectoryClient(hc2)
	updateCl := dirclient.NewUpdateDirectoryClient(hc2)
	return readCl, updateCl, func() {
		svc.Stop()
		_ = store.Close()
		hc2.Disconnect()
		hc1.Disconnect()
	}
}

// generate a JSON serialized TD document
func createTDDoc(thingID string, title string) []byte {
	td := &thing.TD{
		ID:    thingID,
		Title: title,
	}
	tdDoc, _ := json.Marshal(td)
	return tdDoc
}

func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)

	testServer, err = testenv.StartTestServer(core, true)
	serverURL, _, _ = testServer.MsgServer.GetServerURLs()
	if err != nil {
		panic(err)
	}

	res := m.Run()
	testServer.Stop()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop start ---")
	defer t.Log("--- TestStartStop end ---")

	_ = os.Remove(testStoreFile)
	rd, up, stopFunc := startDirectory()
	defer stopFunc()
	assert.NotNil(t, rd)
	assert.NotNil(t, up)

	// viewers should be able to read the directory
	hc, err := testServer.AddConnectClient("user1", authapi.ClientTypeUser, authapi.ClientRoleViewer)
	assert.NoError(t, err)
	defer hc.Disconnect()
	dirCl := dirclient.NewReadDirectoryClient(hc)
	tdList, err := dirCl.GetTDs(0, 10)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	_ = tdList
}

func TestAddRemoveTD(t *testing.T) {
	t.Log("--- TestAddRemoveTD start ---")
	defer t.Log("--- TestAddRemoveTD end ---")
	_ = os.Remove(testStoreFile)
	const agentID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"

	rd, up, stopFunc := startDirectory()
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := up.UpdateTD(agentID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	tv2, err := rd.GetTD(agentID, thing1ID)
	if assert.NoError(t, err) {
		assert.NotNil(t, tv2)
		assert.Equal(t, thing1ID, tv2.ThingID)
		assert.Equal(t, tdDoc1, tv2.Data)
	}
	err = up.RemoveTD(agentID, thing1ID)
	assert.NoError(t, err)
	// after removal, getTD should fail
	td3, err := rd.GetTD(agentID, thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

//func TestListTDs(t *testing.T) {
//	slog.Infof("--- TestListTDs start ---")
//	_ = os.Remove(dirStoreFile)
//	const thing1ID = "thing1"
//	const title1 = "title1"
//
//	ctx := context.Background()
//	store, cancelFunc, err := startDirectory(testUseCapnp)
//	defer cancelFunc()
//	require.NoError(t, err)
//
//	readCap := store.CapReadDirectory(ctx)
//	defer readCap.Release()
//	updateCap := store.CapUpdateDirectory(ctx)
//	defer updateCap.Release()
//	tdDoc1 := createTDDoc(thing1ID, title1)
//
//	err = updateCap.UpdateTD(ctx, thing1ID, tdDoc1)
//	require.NoError(t, err)
//
//	tdList, err := readCap.ListTDs(ctx, 0, 0)
//	require.NoError(t, err)
//	assert.NotNil(t, tdList)
//	assert.True(t, len(tdList) > 0)
//	slog.Infof("--- TestListTDs end ---")
//}

func TestPubTD(t *testing.T) {
	t.Log("--- TestPubTD start ---")
	defer t.Log("--- TestPubTD end ---")
	_ = os.Remove(testStoreFile)
	const agentID = "test"
	const thing1ID = "thing3"
	const title1 = "title1"

	// create a device client that publishes TD documents
	deviceCl, err := testServer.AddConnectClient(agentID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer deviceCl.Disconnect()

	// fire up the directory
	rd, up, stopFunc := startDirectory()
	_ = up
	defer stopFunc()

	// publish a TD
	tdDoc := createTDDoc(thing1ID, title1)
	err = deviceCl.PubEvent(thing1ID, vocab.EventNameTD, tdDoc)
	//err = deviceCl.PubTD(tdDoc) // TODO:create a TD instance

	assert.NoError(t, err)
	time.Sleep(time.Millisecond)

	// expect it to be added to the directory
	tv2, err := rd.GetTD(agentID, thing1ID)
	require.NoError(t, err)
	assert.NotNil(t, tv2)
	assert.Equal(t, thing1ID, tv2.ThingID)
	assert.Equal(t, tdDoc, tv2.Data)
}

func TestCursor(t *testing.T) {
	t.Log("--- TestCursor start ---")
	defer t.Log("--- TestCursor end ---")
	_ = os.Remove(testStoreFile)
	const publisherID = "urn:test"
	const thing1ID = "urn:thing1"
	const title1 = "title1"

	rd, up, stopFunc := startDirectory()
	defer stopFunc()

	// add 1 doc. the service itself also has a doc
	tdDoc1 := createTDDoc(thing1ID, title1)
	err := up.UpdateTD(publisherID, thing1ID, tdDoc1)
	require.NoError(t, err)

	// expect 3 docs, two service capabilities and the one just added
	cursor, err := rd.GetCursor()
	require.NoError(t, err)
	defer cursor.Release()

	tdValue, valid, err := cursor.First()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, tdValue)
	assert.NotEmpty(t, tdValue.Data)

	tdValue, valid, err = cursor.Next() // second
	tdValue, valid, err = cursor.Next() // 3rd
	assert.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, tdValue)
	assert.NotEmpty(t, tdValue.Data)

	tdValue, valid, err = cursor.Next() // there is no 4th
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, tdValue)

	tdValues, valid, err := cursor.NextN(10) // still no third
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, tdValues)
}

//func TestQueryTDs(t *testing.T) {
//	slog.Infof("--- TestQueryTDs start ---")
//	_ = os.Remove(dirStoreFile)
//	const thing1ID = "thing1"
//	const title1 = "title1"
//
//	ctx := context.Background()
//	store, cancelFunc, err := startDirectory(testUseCapnp)
//	defer cancelFunc()
//	require.NoError(t, err)
//	readCap := store.CapReadDirectory(ctx)
//	defer readCap.Release()
//	updateCap := store.CapUpdateDirectory(ctx)
//	defer updateCap.Release()
//
//	tdDoc1 := createTDDoc(thing1ID, title1)
//	err = updateCap.UpdateTD(ctx, thing1ID, tdDoc1)
//	require.NoError(t, err)
//
//	jsonPathQuery := `$[?(@.id=="thing1")]`
//	tdList, err := readCap.QueryTDs(ctx, jsonPathQuery, 0, 0)
//	require.NoError(t, err)
//	assert.NotNil(t, tdList)
//	assert.True(t, len(tdList) > 0)
//	el0 := thing.ThingDescription{}
//	json.Unmarshal([]byte(tdList[0]), &el0)
//	assert.Equal(t, thing1ID, el0.ID)
//	assert.Equal(t, title1, el0.Title)
//	slog.Infof("--- TestQueryTDs end ---")
//}
