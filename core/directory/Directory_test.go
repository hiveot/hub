package directory_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/core/directory/service"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var testFolder = path.Join(os.TempDir(), "test-directory")
var testStoreFile = path.Join(testFolder, "directory.json")
var core = "mqtt"

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

// startDirectory initializes a Directory service, optionally using capnp RPC
func startDirectory() (
	r directory.IReadDirectory, u directory.IUpdateDirectory, stopFn func()) {

	slog.Info("startDirectory start")
	defer slog.Info("startDirectory ended")
	_ = os.Remove(testStoreFile)
	store := kvbtree.NewKVStore(directory.ServiceName, testStoreFile)
	err := store.Open()
	if err != nil {
		panic("unable to open directory store")
	}

	// the directory service needs a server connection
	hc1, err := testServer.AddConnectClient(
		directory.ServiceName, auth.ClientTypeService, auth.ClientRoleService)
	svc := service.NewDirectoryService(store, hc1)
	err = svc.Start()
	if err != nil {
		panic("service fails to start")
	}

	// connect as a user to the server above
	hc2, err := testServer.AddConnectClient(
		"admin", auth.ClientTypeUser, auth.ClientRoleAdmin)
	readCl := dirclient.NewReadDirectoryClient(hc2)
	updateCl := dirclient.NewUpdateDirectoryClient(hc2)
	return readCl, updateCl, func() {
		hc2.Disconnect()
		hc1.Disconnect()
		_ = svc.Stop()
		_ = store.Close()
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

	testServer, err = testenv.StartTestServer(core)
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
}

func TestAddRemoveTD(t *testing.T) {
	t.Log("--- TestAddRemoveTD start ---")
	defer t.Log("--- TestAddRemoveTD end ---")
	_ = os.Remove(testStoreFile)
	const agentID = "urn:test"
	const thing1ID = "urn:thing1"
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

	// expect 2 docs, the service itself and the one just added
	cursor, err := rd.GetCursor()
	assert.NoError(t, err)
	defer cursor.Release()

	tdValue, valid, err := cursor.First()
	assert.True(t, valid)
	assert.NoError(t, err)
	assert.NotEmpty(t, tdValue)
	assert.NotEmpty(t, tdValue.Data)

	tdValue, valid, err = cursor.Next() // second
	assert.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, tdValue)
	assert.NotEmpty(t, tdValue.Data)

	tdValue, valid, err = cursor.Next() // there is no third
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, tdValue)

	tdValues, valid, err := cursor.NextN(10) // still no third
	assert.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, tdValues)
}

func TestPubSub(t *testing.T) {
	t.Log("--- TestPubSub start ---")
	defer t.Log("--- TestPubSub end ---")
	_ = os.Remove(testStoreFile)
	const agentID = "test"
	const thing1ID = "thing3"
	const title1 = "title1"

	// create a device client that publishes TD documents
	deviceCl, err := testServer.AddConnectClient(agentID, auth.ClientTypeDevice, auth.ClientRoleDevice)
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

	// expect it to be added to the directory
	tv2, err := rd.GetTD(agentID, thing1ID)
	require.NoError(t, err)
	assert.NotNil(t, tv2)
	assert.Equal(t, thing1ID, tv2.ThingID)
	assert.Equal(t, tdDoc, tv2.Data)
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

// simple performance test update/read, comparing direct vs capnp access
// TODO: turn into bench test
func TestPerf(t *testing.T) {
	t.Log("--- TestPerf start ---")
	defer t.Log("--- TestPerf end ---")
	_ = os.Remove(testStoreFile)
	const publisherID = "urn:test"
	const thing1ID = "urn:thing1"
	const title1 = "title1"
	const count = 1000

	// fire up the directory
	rd, up, stopFunc := startDirectory()
	_ = up
	defer stopFunc()

	// test update
	t1 := time.Now()
	for i := 0; i < count; i++ {
		tdDoc1 := createTDDoc(thing1ID, title1)
		err := up.UpdateTD(publisherID, thing1ID, tdDoc1)
		require.NoError(t, err)
	}
	d1 := time.Now().Sub(t1)
	fmt.Printf("Duration for update %d iterations: %d msec\n", count, int(d1.Milliseconds()))

	// test read
	t2 := time.Now()
	for i := 0; i < count; i++ {
		td, err := rd.GetTD(publisherID, thing1ID)
		require.NoError(t, err)
		assert.NotNil(t, td)
	}
	d2 := time.Now().Sub(t2)
	fmt.Printf("Duration for read %d iterations: %d msec\n", count, int(d2.Milliseconds()))
}
