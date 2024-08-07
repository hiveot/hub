package state_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/services/state/service"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/services/state/stateclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"
)

var ts *testenv.TestServer

// return an API to the state service
func startStateService(cleanStart bool) (
	svc *service.StateService, stateCl *stateclient.StateClient, stopFn func()) {
	ts = testenv.StartTestServer(cleanStart)

	// the service needs a server connection
	hc1, token1 := ts.AddConnectService(stateapi.AgentID)
	_ = token1

	storeDir := path.Join(ts.TestDir, "test-state")
	svc = service.NewStateService(storeDir)
	err := svc.Start(hc1)

	if err != nil {
		panic("service fails to start: " + err.Error())
	}

	// connect as a user to the service above
	hc2, token2 := ts.AddConnectUser("user1", authn.ClientRoleViewer)
	_ = token2
	stateCl = stateclient.NewStateClient(hc2)
	time.Sleep(time.Millisecond)
	return svc, stateCl, func() {
		hc2.Disconnect()
		slog.Warn("Disconnected " + hc2.ClientID())
		hc1.Disconnect()
		slog.Warn("Disconnected " + hc1.ClientID())
		svc.Stop()
		ts.Stop()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	fmt.Println("--- TestStartStop ---")
	_, stateCl, stopFn := startStateService(true)
	defer stopFn()
	assert.NotNil(t, stateCl)
}

func TestStartStopBadLocation(t *testing.T) {
	fmt.Println("--- TestStartStopBadLocation ---")

	_, _, stopFn := startStateService(true)
	defer stopFn()

	// use a read-only folder
	stateSvc := service.NewStateService("/not/a/folder")
	err := stateSvc.Start(nil)
	require.Error(t, err)

	// stop should not break things further
	stateSvc.Stop()
	assert.Error(t, err)
}

func TestSetGet1(t *testing.T) {
	fmt.Println("--- TestSetGet1 ---")
	const key1 = "key1"
	var val1 = "value 1"
	var val2 = ""
	var val3 = ""

	_, stateCl, stopFn := startStateService(true)
	defer stopFn()

	err := stateCl.Set(key1, val1)
	assert.NoError(t, err)

	found, err := stateCl.Get(key1, &val2)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val2)

	// restart service to check if it persists
	stopFn()
	_, stateCl, stopFn = startStateService(false)
	defer stopFn()
	//
	found, err = stateCl.Get(key1, &val3)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val2)
	assert.Equal(t, val1, val3)

}

func TestSetGetMultiple(t *testing.T) {
	fmt.Println("--- TestSetGetMultiple ---")
	const key1 = "key1"
	const key2 = "key2"
	var val1 = "value 1"
	var val2 = "value 2"
	data := map[string]string{
		key1: val1,
		key2: val2,
	}

	_, stateCl, stopFn := startStateService(true)
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
	fmt.Println("--- TestDelete ---")
	const key1 = "key1"
	var val1 = "value 1"
	var val2 = ""
	var val3 = ""

	_, stateCl, stopFn := startStateService(true)
	defer stopFn()

	// set and get should succeed
	err := stateCl.Set(key1, val1)
	assert.NoError(t, err)
	found, err := stateCl.Get(key1, &val2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, val1, val2)

	// delete should not return an error
	err = stateCl.Delete(key1)
	assert.NoError(t, err)
	// get should return not found without an error
	found, err = stateCl.Get(key1, &val3)
	assert.NoError(t, err)
	assert.False(t, found)
	assert.NotEqual(t, val1, val3)

	multiple, err := stateCl.GetMultiple([]string{key1})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(multiple))

}

func TestGetDifferentClientBuckets(t *testing.T) {
	fmt.Println("--- TestGetDifferentClientBuckets ---")
	const clientID1 = "test-client1"
	const clientID2 = "test-client2"
	const key1 = "key1"
	const key2 = "key2"
	var val1 = "value 1"
	var val2 = "value 2"

	_, stateCl, stopFn := startStateService(true)
	_ = stateCl
	defer stopFn()

	hc1, token1 := ts.AddConnectAgent(clientID1)
	require.NotEmpty(t, token1)
	defer hc1.Disconnect()
	hc2, token2 := ts.AddConnectService(clientID2)
	require.NotEmpty(t, token2)
	defer hc2.Disconnect()

	// both clients set a record
	cl1 := stateclient.NewStateClient(hc1)
	cl2 := stateclient.NewStateClient(hc2)

	err := cl1.Set(key1, val1)
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
//	fmt.Println("--- TestCursor ---")
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
