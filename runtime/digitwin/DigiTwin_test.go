package digitwin_test

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"
)

var testFolder = path.Join(os.TempDir(), "test-directory")
var storePath = path.Join(testFolder, "digitwin.json")
var valueNames = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

// startService initializes a service and a client
// This doesn't use any transport.
func startService(clean bool) (
	svc *service.DigiTwinService,
	cl *digitwinclient.DigiTwinClient,
	stopFn func()) {

	if clean {
		_ = os.Remove(storePath)
	}
	store := kvbtree.NewKVStore(storePath)
	err := store.Open()
	if err != nil {
		panic("unable to open the digital twin bucket store")
	}

	svc = service.NewDigiTwinService(store)
	err = svc.Start()
	if err != nil {
		panic("unable to start the digital twin service")
	}

	// create the RPC communication layer
	svcRPC := msghandler.NewDigiTwinRPC(svc)
	pm := func(thingID string, method string, args interface{}, reply interface{}) error {
		// message transport passes it straight to the service rpc handler
		data, _ := json.Marshal(args)
		tm := things.NewThingMessage(vocab.MessageTypeAction,
			thingID, method, data, "testclient")
		resp, err := svcRPC.HandleMessage(tm)
		if reply != nil && resp != nil {
			err = json.Unmarshal(resp, &reply)
		}
		return err
	}
	cl = digitwinclient.NewDigiTwinClient(pm)

	return svc, cl, func() {
		svc.Stop()
		store.Close()
	}
}

// generate a JSON serialized TD document
func createTDDoc(thingID string, title string) *things.TD {
	td := things.NewTD(thingID, title, vocab.ThingDevice)
	return td
}

// generate a random batch of values for testing
func createValueBatch(svc *service.DigiTwinService,
	nrValues int, thingIDs []string, timespanSec int) (batch []*things.ThingMessage) {

	valueBatch := make([]*things.ThingMessage, 0, nrValues)
	for j := 0; j < nrValues; j++ {
		thingIndex := rand.Intn(len(thingIDs))
		thingID := thingIDs[thingIndex]
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		randomTime := time.Now().Add(-randomSeconds)

		ev := things.NewThingMessage(vocab.MessageTypeEvent,
			thingID, valueNames[randomName],
			[]byte(fmt.Sprintf("%2.3f", randomValue)), "sender1",
		)
		ev.CreatedMSec = randomTime.UnixMilli()

		_ = svc.Values.HandleEvent(ev)
		valueBatch = append(valueBatch, ev)
	}
	return valueBatch
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
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, cl, stopFunc := startService(true)

	// add TDs and values
	for _, thingID := range thingIDs {
		td := createTDDoc(thingID, thingID)
		svc.Directory.UpdateThing("test", thingID, td)
	}
	createValueBatch(svc, 100, thingIDs, 100)

	// viewers should be able to read the directory
	tdList, err := svc.Directory.ReadThings(0, 10)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	assert.Equal(t, len(thingIDs), len(tdList))
	valList1, err := cl.ReadProperties(thingIDs[1], nil)
	assert.NoError(t, err)
	assert.True(t, len(valList1) > 1)

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, cl, stopFunc = startService(false)
	defer stopFunc()
	tdList2, err := cl.ReadThings(0, 10)
	assert.Equal(t, len(thingIDs), len(tdList2))

	valList2a, err := cl.ReadProperties(thingIDs[1], nil)
	assert.NoError(t, err)
	assert.Equal(t, len(valList1), len(valList2a))
	valList2b, err := cl.ReadEvents(thingIDs[1], nil)
	_ = valList2b
	assert.NoError(t, err)
	valList2c, err := cl.ReadActions(thingIDs[1], nil)
	_ = valList2c
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	const count = 100
	const agent1ID = "agent1"
	const thing1ID = "thing1" // matches a percentage of the random things

	svc, cl, closeFn := startService(true)
	defer closeFn()

	batch := createValueBatch(svc, count, []string{thing1ID}, 3600*24*30)
	_ = batch

	t0 := time.Now()

	values, err := cl.ReadProperties(thing1ID, nil)
	require.NoError(t, err)
	require.NotNil(t, values)
	d0 := time.Now().Sub(t0)

	// 2nd time from cache
	t1 := time.Now()
	values2, err := cl.ReadProperties(thing1ID, nil)
	require.NoError(t, err)
	require.NotNil(t, values2)
	d1 := time.Now().Sub(t1)

	assert.Less(t, d1, d0)

	values3, err := cl.ReadProperties(thing1ID, valueNames)
	require.NoError(t, err)
	require.NotNil(t, values3)
	_ = values3

	// save and reload props
	svc.Stop()
	err = svc.Start()

	assert.NoError(t, err)
	found := svc.Values.LoadValues(thing1ID)
	assert.False(t, found) // not cached
}

func TestAddPropsEvent(t *testing.T) {
	thing1ID := "thing1"
	pev := make(map[string]string)
	pev["temperature"] = "10"
	pev["humidity"] = "33"
	pev["switch"] = "false"
	serProps, _ := json.Marshal(pev)

	svc, cl, closeFn := startService(true)
	defer closeFn()

	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, serProps, "sender")
	_ = svc.Values.HandleEvent(tv)

	values1, err := cl.ReadProperties(thing1ID, valueNames)
	require.NoError(t, err)
	assert.Equal(t, len(pev), len(values1))
}

func TestAddPropsFail(t *testing.T) {
	thing1ID := "badthingid"
	svc, cl, closeFn := startService(true)
	_ = svc
	defer closeFn()
	values1, err := cl.ReadProperties(thing1ID, valueNames)
	require.NoError(t, err)
	assert.Empty(t, values1)
}

func TestAddBadProps(t *testing.T) {
	thing1ID := "thing1"
	badProps := []string{"bad1", "bad2"}
	serProps, _ := json.Marshal(badProps)

	svc, cl, closeFn := startService(true)
	defer closeFn()
	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, serProps, "sender")
	err := svc.Values.HandleEvent(tv)
	assert.Error(t, err)

	//// action is ignored
	//tv.MessageType = vocab.MessageTypeAction
	//_, err = svc.HandleEvent(tv)
	//assert.NoError(t, err)
	//
	values1, err := cl.ReadProperties(thing1ID, valueNames)
	require.NoError(t, err)
	assert.Equal(t, 0, len(values1))
}

func TestAddRemoveTD(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, cl, stopFunc := startService(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := svc.Directory.UpdateThing(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	td2, err := cl.ReadThing(thing1ID)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

	// after removal, getTD should return nil
	err = cl.RemoveThing(thing1ID)
	assert.NoError(t, err)

	td3, err := cl.ReadThing(thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleEvent(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, cl, stopFunc := startService(true)
	dirRPC := msghandler.NewDigiTwinRPC(svc)
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

	tdList, err := cl.ReadThings(0, 10)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, cl, stopFunc := startService(true)
	_ = svc
	defer stopFunc()
	tds, err := cl.ReadThings(0, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	tds, err = cl.ReadThings(10, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	// bad clientID
	tdd1, err := cl.ReadThing("badid")
	require.Error(t, err)
	require.Nil(t, tdd1)
}

func TestListTDs(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, cl, stopFunc := startService(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)

	err := svc.Directory.UpdateThing(senderID, thing1ID, tdDoc1)
	require.NoError(t, err)

	tdList, err := cl.ReadThings(0, 10)
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
