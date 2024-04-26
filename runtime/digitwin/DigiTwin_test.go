package digitwin_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"github.com/hiveot/hub/runtime/digitwin/digitwinhandler"
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
	mt api.IMessageTransport,
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

	// create the messaging wrapper
	msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	mt = func(thingID string, method string, args interface{}, reply interface{}) error {
		// message transport passes it straight to the service rpc handler
		data, _ := json.Marshal(args)
		tm := things.NewThingMessage(vocab.MessageTypeAction,
			thingID, method, data, "testclient")
		resp, err := msgHandler(tm)
		if reply != nil && resp != nil {
			err = json.Unmarshal(resp, &reply)
		}
		return err
	}

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

		_ = svc.Values.HandleMessage(ev)
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
	svc, mt, stopFunc := startService(true)

	// add TDs and values
	for _, thingID := range thingIDs {
		td := createTDDoc(thingID, thingID)
		err := svc.Directory.UpdateThing("test", thingID, td)
		require.NoError(t, err)
	}
	createValueBatch(svc, 100, thingIDs, 100)

	// viewers should be able to read the directory
	tdList, err := svc.Directory.ReadThings(0, 10)
	assert.NoError(t, err, "Cant read directory. Did the service set client permissions?")
	assert.Equal(t, len(thingIDs), len(tdList))
	valList1, err := digitwinclient.ReadEvents(mt, thingIDs[1], nil, "")
	//valList1, err := cl.ReadProperties(thingIDs[1], nil)
	assert.NoError(t, err)
	assert.True(t, len(valList1) > 1)

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, mt, stopFunc = startService(false)
	defer stopFunc()
	tdList2, err := digitwinclient.ReadThings(mt, 0, 10)
	assert.Equal(t, len(thingIDs), len(tdList2))

	valList2a, err := digitwinclient.ReadProperties(mt, thingIDs[1], nil, "")
	assert.NoError(t, err)
	assert.Equal(t, len(valList1), len(valList2a))
	valList2b, err := digitwinclient.ReadEvents(mt, thingIDs[1], nil, "")
	_ = valList2b
	assert.NoError(t, err)
	valList2c, err := digitwinclient.ReadActions(mt, thingIDs[1], nil, "")
	_ = valList2c
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	const count = 100
	const agent1ID = "agent1"
	const thing1ID = "thing1" // matches a percentage of the random things

	svc, mt, closeFn := startService(true)
	defer closeFn()

	batch := createValueBatch(svc, count, []string{thing1ID}, 3600*24*30)
	_ = batch

	t0 := time.Now()

	values, err := digitwinclient.ReadProperties(mt, thing1ID, nil, "")
	require.NoError(t, err)
	require.NotNil(t, values)
	d0 := time.Now().Sub(t0)

	// 2nd time from cache
	t1 := time.Now()
	values2, err := digitwinclient.ReadProperties(mt, thing1ID, nil, "")
	require.NoError(t, err)
	require.NotNil(t, values2)
	d1 := time.Now().Sub(t1)

	assert.Less(t, d1, d0)

	values3, err := digitwinclient.ReadProperties(mt, thing1ID, valueNames, "")
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

	svc, mt, closeFn := startService(true)
	defer closeFn()

	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, serProps, "sender")
	_ = svc.Values.HandleMessage(tv)

	values1, err := digitwinclient.ReadProperties(mt, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Equal(t, len(pev), len(values1))
}

func TestAddPropsFail(t *testing.T) {
	thing1ID := "badthingid"
	svc, mt, closeFn := startService(true)
	_ = svc
	defer closeFn()
	values1, err := digitwinclient.ReadProperties(mt, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Empty(t, values1)
}

func TestAddBadProps(t *testing.T) {
	thing1ID := "thing1"
	badProps := []string{"bad1", "bad2"}
	serProps, _ := json.Marshal(badProps)

	svc, mt, closeFn := startService(true)
	defer closeFn()
	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, serProps, "sender")
	err := svc.Values.HandleMessage(tv)
	assert.Error(t, err)

	//// action is ignored
	//tv.MessageType = vocab.MessageTypeAction
	//_, err = svc.HandleEvent(tv)
	//assert.NoError(t, err)
	//
	values1, err := digitwinclient.ReadProperties(mt, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Equal(t, 0, len(values1))
}

func TestAddRemoveTD(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startService(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)
	err := svc.Directory.UpdateThing(senderID, thing1ID, tdDoc1)
	assert.NoError(t, err)

	td2, err := digitwinclient.ReadThing(mt, thing1ID)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, td2.ID)

	// after removal, getTD should return nil
	err = digitwinclient.RemoveThing(mt, thing1ID)
	assert.NoError(t, err)

	td3, err := digitwinclient.ReadThing(mt, thing1ID)
	assert.Empty(t, td3)
	assert.Error(t, err)
}

func TestHandleTDEvent(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startService(true)
	msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	defer stopFunc()

	// events should be handled
	tdDoc1 := createTDDoc(thing1ID, title1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	tv := things.NewThingMessage(vocab.MessageTypeEvent, thing1ID,
		vocab.EventTypeTD, tdDoc1Json, senderID)
	_, err := msgHandler(tv)
	assert.NoError(t, err)

	// non-events like actions should be ignored
	tv.MessageType = vocab.MessageTypeAction
	_, err = msgHandler(tv)
	assert.NoError(t, err)

	tdList, err := digitwinclient.ReadThings(mt, 0, 10)
	assert.Equal(t, 1, len(tdList))
	assert.NoError(t, err)
}

func TestGetTDsFail(t *testing.T) {
	const clientID = "client1"
	svc, mt, stopFunc := startService(true)
	_ = svc
	defer stopFunc()
	tds, err := digitwinclient.ReadThings(mt, 0, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	tds, err = digitwinclient.ReadThings(mt, 10, 10)
	require.NoError(t, err)
	require.Empty(t, tds)

	// bad clientID
	tdd1, err := digitwinclient.ReadThing(mt, "badid")
	require.Error(t, err)
	require.Nil(t, tdd1)
}

func TestListTDs(t *testing.T) {
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	svc, mt, stopFunc := startService(true)
	defer stopFunc()

	tdDoc1 := createTDDoc(thing1ID, title1)

	err := svc.Directory.UpdateThing(senderID, thing1ID, tdDoc1)
	require.NoError(t, err)

	tdList, err := digitwinclient.ReadThings(mt, 0, 10)
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
