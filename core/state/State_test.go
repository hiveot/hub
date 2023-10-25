package state_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/state/service"
	"github.com/hiveot/hub/core/state/stateapi"
	"github.com/hiveot/hub/core/state/stateclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
)

const storeDir = "/tmp/test-state"
const core = "mqtt"

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

// return an API to the state service
func startStateService(cleanStart bool) (stateCl *stateclient.StateClient, stopFn func()) {
	slog.Info("startStateService")
	if cleanStart {
		_ = os.RemoveAll(storeDir)
	}
	// the service needs a server connection
	hc1, err := testServer.AddConnectClient(
		stateapi.ServiceName, authapi.ClientTypeService, authapi.ClientRoleService)

	svc := service.NewStateService(hc1, storeDir)
	err = svc.Start()
	if err != nil {
		panic("service fails to start: " + err.Error())
	}

	// connect as a user to the service above
	hc2, err := testServer.AddConnectClient(
		"user1", authapi.ClientTypeUser, authapi.ClientRoleViewer)
	stateCl = stateclient.NewStateClient(hc2)
	return stateCl, func() {
		hc2.Disconnect()
		svc.Stop()
		hc1.Disconnect()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	var err error
	testServer, err = testenv.StartTestServer(core, true)
	serverURL, _, _ = testServer.MsgServer.GetServerURLs()
	if err != nil {
		panic(err)
	}

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")
	stateCl, stopFn := startStateService(true)
	assert.NotNil(t, stateCl)

	stopFn()
}

func TestStartStopBadLocation(t *testing.T) {
	t.Log("--- TestStartStopBadLocation ---")

	// the service needs a server connection
	hc1, err := testServer.AddConnectClient(
		stateapi.ServiceName, authapi.ClientTypeService, authapi.ClientRoleService)

	// use a read-only folder
	stateSvc := service.NewStateService(hc1, "/not/a/folder")
	err = stateSvc.Start()
	require.Error(t, err)

	// stop should not break things further
	stateSvc.Stop()
	assert.Error(t, err)
}

func TestSetGet1(t *testing.T) {
	t.Log("--- TestSetGet1 ---")
	const clientID1 = "test-client1"
	const appID = "test-app"
	const key1 = "key1"
	var val1 = "value 1"
	var val2 = ""
	var val3 = ""

	stateCl, stopFn := startStateService(true)

	err := stateCl.Set(key1, val1)
	assert.NoError(t, err)

	found, err := stateCl.Get(key1, &val2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val2)

	// check if it persists
	stopFn()
	stateCl, stopFn = startStateService(false)
	found, err = stateCl.Get(key1, &val3)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val3)

}

func TestSetGetMultiple(t *testing.T) {
	t.Log("--- TestSetGetMultiple ---")
	const clientID1 = "test-client1"
	const appID = "test-app"
	const key1 = "key1"
	const key2 = "key2"
	var val1 = []byte("value 1")
	var val2 = []byte("value 2")
	data := map[string][]byte{
		key1: val1,
		key2: val2,
	}

	stateCl, stopFn := startStateService(true)
	defer stopFn()

	// write multiple
	err := stateCl.SetMultiple(data)
	assert.NoError(t, err)

	// result must match
	data2, err := stateCl.GetMultiple([]string{key1, key2})
	_ = data2
	assert.NoError(t, err)
	assert.Equal(t, data[key1], data2[key1])
}

func TestDelete(t *testing.T) {
	t.Log("--- TestDelete ---")
	const clientID1 = "test-client1"
	const appID = "test-app"
	const key1 = "key1"
	var val1 = "value 1"
	var val2 = ""
	var val3 = ""

	stateCl, stopFn := startStateService(true)
	defer stopFn()

	// set and get should succeed
	err := stateCl.Set(key1, val1)
	assert.NoError(t, err)
	found, err := stateCl.Get(key1, &val2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val2)

	// delete and get should not return a value
	err = stateCl.Delete(key1)
	assert.NoError(t, err)
	found, err = stateCl.Get(key1, &val3)
	assert.NoError(t, err)
	assert.False(t, found)
	assert.NotEqual(t, val1, val3)

	multiple, err := stateCl.GetMultiple([]string{key1})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(multiple))

}

func TestGetDifferentClientBuckets(t *testing.T) {
	t.Log("--- TestGetDifferentClientBuckets ---")
	const clientID1 = "test-client1"
	const clientID2 = "test-client2"
	const appID = "test-app"
	const key1 = "key1"
	const key2 = "key2"
	var val1 = "value 1"
	var val2 = "value 2"

	hc1, err := testServer.AddConnectClient(clientID1, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc1.Disconnect()
	hc2, err := testServer.AddConnectClient(clientID2, authapi.ClientTypeService, authapi.ClientRoleService)
	require.NoError(t, err)
	defer hc2.Disconnect()
	_, stopFn := startStateService(true)
	defer stopFn()

	// both clients set a record
	cl1 := stateclient.NewStateClient(hc1)
	cl2 := stateclient.NewStateClient(hc2)

	err = cl1.Set(key1, val1)
	assert.NoError(t, err)
	err = cl2.Set(key2, val2)

	// clients cannot read the other's value
	tmp1 := ""
	tmp2 := ""
	found1, err := cl1.Get(key2, &tmp1)
	assert.NoError(t, err)
	assert.False(t, found1)
	found2, err := cl2.Get(key1, &tmp2)
	assert.NoError(t, err)
	assert.False(t, found2)
}

//func TestCursor(t *testing.T) {
//	t.Log("--- TestCursor ---")
//	const clientID1 = "test-client1"
//	const appID = "test-app"
//	const key1 = "key1"
//	const key2 = "key2"
//	var val1 = []byte("value 1")
//	var val2 = []byte("value 2")
//	data := map[string][]byte{
//		key1: val1,
//		key2: val2,
//	}
//
//	ctx := context.Background()
//	svc, stopFn, err := startStateService(testUseCapnp)
//	defer stopFn()
//	clientState, _ := svc.CapClientState(ctx, clientID1, appID)
//
//	// write multiple
//	err = clientState.SetMultiple(ctx, data)
//	assert.NoError(t, err)
//
//	// result must match
//	cursor := clientState.Cursor(ctx)
//	assert.NotNil(t, cursor)
//	k1, v, valid := cursor.First()
//	assert.True(t, valid)
//	assert.NotNil(t, k1)
//	assert.Equal(t, val1, v)
//	k0, _, valid := cursor.Prev()
//	assert.False(t, valid)
//	assert.Empty(t, k0)
//	k2, v, valid := cursor.Seek(key1)
//	assert.True(t, valid)
//	assert.Equal(t, key1, k2)
//	assert.Equal(t, val1, v)
//	k3, _, valid := cursor.Next()
//	assert.True(t, valid)
//	assert.Equal(t, key2, k3)
//	k4, _, valid := cursor.Last()
//	assert.True(t, valid)
//	assert.Equal(t, key2, k4)
//	//
//	cursor.Release()
//
//	// cleanup
//	clientState.Release()
//
//}
