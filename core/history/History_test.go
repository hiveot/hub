package history_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/history/config"
	"github.com/hiveot/hub/core/history/historyapi"
	"github.com/hiveot/hub/core/history/historyclient"
	"github.com/hiveot/hub/core/history/service"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"

	"github.com/hiveot/hub/lib/thing"
)

const thingIDPrefix = "thing-"

// when testing using the capnp RPC
var testFolder = path.Join(os.TempDir(), "test-history")

const core = "nats"

// recommended store for history is pebble
const historyStoreBackend = buckets.BackendPebble

//const historyStoreBackend = buckets.BackendBBolt
//const historyStoreBackend = buckets.BackendKVBTree

const serviceID = historyapi.ServiceName
const testClientID = "operator1"

// the following are set by the testmain
var testServer *testenv.TestServer

var names = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

// Create a new store, delete if it already exists
func newHistoryService() (
	//store buckets.IBucketStore,
	svc *service.HistoryService,
	r *historyclient.ReadHistoryClient,
	stopFn func()) {

	svcConfig := config.NewHistoryConfig(testFolder)

	// create a new empty store to use
	_ = os.RemoveAll(svcConfig.StoreDirectory)
	histStore := bucketstore.NewBucketStore(testFolder, serviceID, historyStoreBackend)
	err := histStore.Open()
	if err != nil {
		panic("can't open history bucket store")
	}

	// the service needs a server connection
	hc, err := testServer.AddConnectClient(historyapi.ServiceName, authapi.ClientTypeService, authapi.ClientRoleService)
	svc = service.NewHistoryService(histStore)
	if err == nil {
		err = svc.Start(hc)
	}
	if err != nil {
		panic("Failed starting the state service: " + err.Error())
	}

	// create an end user client for testing
	hc2, err := testServer.AddConnectClient(testClientID, authapi.ClientTypeUser, authapi.ClientRoleOperator)
	if err != nil {
		panic("can't connect operator")
	}
	histCl := historyclient.NewReadHistoryClient(hc2)

	return svc, histCl, func() {
		hc2.Disconnect()
		svc.Stop()
		_ = histStore.Close()
		hc.Disconnect()
		// give it some time to shut down before the next test
		time.Sleep(time.Millisecond)
	}
}

//func stopStore(store client.IHistory) error {
//	return store.(*mongohs.MongoHistoryServer).Stop()
//}

// generate a random batch of values for testing
func makeValueBatch(publisherID string, nrValues, nrThings, timespanSec int) (
	batch []*thing.ThingValue, highest map[string]*thing.ThingValue) {

	highest = make(map[string]*thing.ThingValue)
	valueBatch := make([]*thing.ThingValue, 0, nrValues)
	for j := 0; j < nrValues; j++ {
		randomID := rand.Intn(nrThings)
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		randomTime := time.Now().Add(-randomSeconds)
		thingID := thingIDPrefix + strconv.Itoa(randomID)

		ev := &thing.ThingValue{
			AgentID:     publisherID,
			ThingID:     thingID,
			Name:        names[randomName],
			Data:        []byte(fmt.Sprintf("%2.3f", randomValue)),
			CreatedMSec: randomTime.UnixMilli(),
		}
		// track the actual most recent event for the name for thing 3
		if randomID == 0 {
			if _, exists := highest[ev.Name]; !exists ||
				highest[ev.Name].CreatedMSec < ev.CreatedMSec {
				highest[ev.Name] = ev
			}
		}
		valueBatch = append(valueBatch, ev)
	}
	return valueBatch, highest
}

// add some history to the store using publisher 'device1'
func addBulkHistory(svc *service.HistoryService, count int, nrThings int, timespanSec int) (highest map[string]*thing.ThingValue) {

	const publisherID = "device1"
	var batchSize = 1000
	if batchSize > count {
		batchSize = count
	}

	evBatch, highest := makeValueBatch(publisherID, count, nrThings, timespanSec)
	addHist := svc.GetAddHistory()

	// use add multiple in 100's
	for i := 0; i < count/batchSize; i++ {
		// no thingID constraint allows adding events from any thing
		start := batchSize * i
		end := batchSize * (i + 1)
		err := addHist.AddEvents(evBatch[start:end])
		if err != nil {
			slog.Error("Problem adding events: %s", err)
		}
	}
	return highest
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)
	testServer, _ = testenv.StartTestServer(core, true)

	res := m.Run()
	os.Exit(res)
}

// Test creating and deleting the history database
// This requires a local unsecured MongoDB instance
func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")

	store, readHist, stopFn := newHistoryService()
	defer stopFn()
	_ = store
	_ = readHist
}

func TestAddGetEvent(t *testing.T) {
	t.Log("--- TestAddGetEvent ---")
	const id1 = "thing1"
	const id2 = "thing2"
	const agent1ID = "device1"
	const thing1ID = id1
	const thing2ID = id2
	const evTemperature = "temperature"
	const evHumidity = "humidity"

	// add history with two specific things to test against
	svc, readHist, cancelFn := newHistoryService()
	// do not defer cancel as it will be closed and reopened in the test
	fivemago := time.Now().Add(-time.Minute * 5)
	fiftyfivemago := time.Now().Add(-time.Minute * 55)
	addBulkHistory(svc, 20, 3, 3600)

	// add thing1 temperature from 5 minutes ago
	addHist := svc.GetAddHistory()
	ev1_1 := &thing.ThingValue{AgentID: agent1ID, ThingID: thing1ID, Name: evTemperature,
		Data: []byte("12.5"), CreatedMSec: fivemago.UnixMilli()}
	err := addHist.AddEvent(ev1_1)
	assert.NoError(t, err)
	// add thing1 humidity from 55 minutes ago
	ev1_2 := &thing.ThingValue{AgentID: agent1ID, ThingID: thing1ID, Name: evHumidity,
		Data: []byte("70"), CreatedMSec: fiftyfivemago.UnixMilli()}
	err = addHist.AddEvent(ev1_2)
	assert.NoError(t, err)

	// add thing2 humidity from 5 minutes ago
	ev2_1 := &thing.ThingValue{AgentID: agent1ID, ThingID: thing2ID, Name: evHumidity,
		Data: []byte("50"), CreatedMSec: fivemago.UnixMilli()}
	err = addHist.AddEvent(ev2_1)
	assert.NoError(t, err)

	// add thing2 temperature from 55 minutes ago
	ev2_2 := &thing.ThingValue{AgentID: agent1ID, ThingID: thing2ID, Name: evTemperature,
		Data: []byte("17.5"), CreatedMSec: fiftyfivemago.UnixMilli()}
	err = addHist.AddEvent(ev2_2)
	assert.NoError(t, err)

	// Test 1: get events of thing1 older than 300 minutes ago - expect 1 humidity from 55 minutes ago
	cursor1, c1Release, err := readHist.GetCursor(agent1ID, thing1ID, "")

	// seek must return the thing humidity added 55 minutes ago, not 5 minutes ago
	timeAfter := time.Now().Add(-time.Minute * 300).UnixMilli()
	tv1, valid, err := cursor1.Seek(timeAfter)
	if assert.NoError(t, err) && assert.True(t, valid) {
		assert.Equal(t, thing1ID, tv1.ThingID)
		assert.Equal(t, evHumidity, tv1.Name)
		// next finds the temperature from 5 minutes ago
		tv2, valid, err := cursor1.Next()
		assert.NoError(t, err)
		if assert.True(t, valid) {
			assert.Equal(t, evTemperature, tv2.Name)
		}
	}

	// Test 2: get events of thing 1 newer than 30 minutes ago - expect 1 temperature
	timeAfter = time.Now().Add(-time.Minute * 30).UnixMilli()

	// do we need to get a new cursor?
	//readHistory = svc.CapReadHistory()
	tv3, valid, _ := cursor1.Seek(timeAfter)
	if assert.True(t, valid) {
		assert.Equal(t, thing1ID, tv3.ThingID)   // must match the filtered id1
		assert.Equal(t, evTemperature, tv3.Name) // must match evTemperature from 5 minutes ago
		assert.Equal(t, fivemago.UnixMilli(), tv3.CreatedMSec)
	}
	c1Release()
	// Stop the service before phase 2
	cancelFn()

	// PHASE 2: after closing and reopening the svc the event should still be there
	store2 := bucketstore.NewBucketStore(testFolder, serviceID, historyStoreBackend)
	err = store2.Open()
	require.NoError(t, err)
	defer store2.Close()
	hc, err := testServer.AddConnectClient(historyapi.ServiceName, authapi.ClientTypeService, authapi.ClientRoleService)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc = service.NewHistoryService(store2)
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// Test 3: get first temperature of thing 2 - expect 1 result
	time.Sleep(time.Second)
	readHist2 := historyclient.NewReadHistoryClient(hc) // reuse hc, its okay
	cursor2, releaseFn, err := readHist2.GetCursor(agent1ID, thing2ID, "")
	require.NoError(t, err)
	tv4, valid, _ := cursor2.First()
	require.True(t, valid)
	assert.Equal(t, evTemperature, tv4.Name)
	releaseFn()
}

func TestAddPropertiesEvent(t *testing.T) {
	t.Log("--- TestAddPropertiesEvent ---")
	//const clientID = "device0"
	const thing1ID = thingIDPrefix + "0" // matches a percentage of the random things
	const agent1 = "device1"
	const temp1 = "55"

	svc, readHist, closeFn := newHistoryService()
	defer closeFn()

	action1 := &thing.ThingValue{
		AgentID: agent1,
		ThingID: thing1ID,
		Name:    vocab.VocabSwitch,
		Data:    []byte("on"),
	}
	event1 := &thing.ThingValue{
		AgentID: agent1,
		ThingID: thing1ID,
		Name:    vocab.VocabTemperature,
		Data:    []byte(temp1),
	}
	badEvent1 := &thing.ThingValue{
		AgentID: agent1,
		ThingID: thing1ID,
		Name:    "", // missing name
	}
	badEvent2 := &thing.ThingValue{
		AgentID: "", // missing publisher
		ThingID: thing1ID,
		Name:    "name",
	}
	badEvent3 := &thing.ThingValue{
		AgentID:     agent1,
		ThingID:     thing1ID,
		Name:        "baddate",
		CreatedMSec: -1,
	}
	badEvent4 := &thing.ThingValue{
		AgentID: agent1,
		ThingID: "", // missing ID
		Name:    "temperature",
	}
	propsList := make(map[string][]byte)
	propsList[vocab.VocabBatteryLevel] = []byte("50")
	propsList[vocab.VocabCPULevel] = []byte("30")
	propsList[vocab.VocabSwitch] = []byte("off")
	propsValue, _ := json.Marshal(propsList)
	props1 := &thing.ThingValue{
		AgentID: agent1,
		ThingID: thing1ID,
		Name:    vocab.EventNameProps,
		Data:    propsValue,
	}

	// in total add 5 properties
	addHist := svc.GetAddHistory()
	err := addHist.AddAction(action1)
	assert.NoError(t, err)
	err = addHist.AddEvent(event1)
	assert.NoError(t, err)
	err = addHist.AddEvent(props1) // props has 3 values
	assert.NoError(t, err)

	// and some bad values
	err = addHist.AddEvent(badEvent1)
	assert.Error(t, err)
	err = addHist.AddEvent(badEvent2)
	assert.Error(t, err)
	err = addHist.AddEvent(badEvent3) // bad date is recovered
	assert.NoError(t, err)
	err = addHist.AddEvent(badEvent4)
	assert.Error(t, err)
	err = addHist.AddAction(badEvent1)
	assert.Error(t, err)

	// verify named properties from different sources
	props, err := readHist.GetLatest(agent1, thing1ID,
		[]string{vocab.VocabTemperature, vocab.VocabSwitch})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(props))
	assert.Equal(t, vocab.VocabTemperature, props[0].Name)
	assert.Equal(t, []byte(temp1), props[0].Data)
	assert.Equal(t, vocab.VocabSwitch, props[1].Name)

}

func TestGetLatest(t *testing.T) {
	t.Log("--- TestGetLatest ---")
	const count = 1000
	const agent1ID = "device1"
	const thing1ID = thingIDPrefix + "0" // matches a percentage of the random things

	store, readHist, closeFn := newHistoryService()
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	highestFromAdded := addBulkHistory(store, count, 1, 3600*24*30)

	values, err := readHist.GetLatest(agent1ID, thing1ID, nil)
	require.NoError(t, err)
	assert.NotNil(t, values)

	cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing1ID, "")
	defer releaseFn()

	t.Logf("Received %d values", len(values))
	assert.Greater(t, len(values), 0, "Expected multiple properties, got none")
	// compare the results with the highest value tracked during creation of the test data
	for _, val := range values {
		t.Logf("Result; name '%s'; created: %d", val.Name, val.CreatedMSec)
		highest := highestFromAdded[val.Name]
		if assert.NotNil(t, highest) {
			t.Logf("Expect %s: %v", highest.Name, highest.CreatedMSec)
			assert.Equal(t, highest.CreatedMSec, val.CreatedMSec)
		}
	}
	// getting the Last should get the same result
	lastItem, valid, _ := cursor.Last()
	highest := highestFromAdded[lastItem.Name]

	assert.True(t, valid)
	assert.Equal(t, lastItem.CreatedMSec, highest.CreatedMSec)
}

func TestPrevNext(t *testing.T) {
	t.Log("--- TestPrevNext ---")
	const count = 1000
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things
	const publisherID = "device1"
	store, readHist, closeFn := newHistoryService()
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(store, count, 1, 3600*24*30)

	cursor, releaseFn, _ := readHist.GetCursor(publisherID, thing0ID, "")
	defer releaseFn()
	assert.NotNil(t, cursor)

	// go forward
	item0, valid, err := cursor.First()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item0)
	item1, valid, err := cursor.Next()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item1)
	items2to11, itemsRemaining, err := cursor.NextN(10)
	require.NoError(t, err)
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))

	// go backwards
	item10to1, itemsRemaining, err := cursor.PrevN(10)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := cursor.Prev()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item0.CreatedMSec, item0b.CreatedMSec)

	// can't skip before the beginning of time
	iteminv, valid, err := cursor.Prev()
	require.NoError(t, err)
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	item11b, valid, err := cursor.Seek(item11.CreatedMSec)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item11.Name, item11b.Name)
}

// filter on property name
func TestPrevNextFiltered(t *testing.T) {
	slog.Info("--- TestPrevNextFiltered ---")
	const count = 1000
	const agent1ID = "device1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things

	svc, readHist, closeFn := newHistoryService()
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(svc, count, 1, 3600*24*30)
	propName := names[2] // names was used to generate the history

	// the latest of propName should be a value containing propName
	values, err := readHist.GetLatest(agent1ID, thing0ID, []string{propName})
	require.NoError(t, err)
	require.Greater(t, len(values), 0)
	assert.Equal(t, propName, values[0].Name)

	// A cursor with a filter on propName should only return results of propName
	cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing0ID, propName)
	require.NoError(t, err)
	defer releaseFn()
	assert.NotNil(t, cursor)
	item0, valid, err := cursor.First()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, propName, item0.Name)

	// further steps should still only return propName
	item1, valid, err := cursor.Next()
	assert.True(t, valid)
	assert.Equal(t, propName, item1.Name)
	items2to11, itemsRemaining, err := cursor.NextN(10)
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))
	assert.Equal(t, propName, items2to11[9].Name)

	// go backwards
	item10to1, itemsRemaining, err := cursor.PrevN(10)
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := cursor.Prev()
	assert.True(t, valid)
	assert.Equal(t, item0.CreatedMSec, item0b.CreatedMSec)
	assert.Equal(t, propName, item0b.Name)

	// can't skip before the beginning of time
	iteminv, valid, err := cursor.Prev()
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	item11b, valid, err := cursor.Seek(item11.CreatedMSec)
	assert.True(t, valid)
	assert.Equal(t, item11.Name, item11b.Name)

	// last item should be of the name
	lastItem, valid, err := cursor.Last()
	assert.True(t, valid)
	assert.Equal(t, propName, lastItem.Name)

	cursor.Release()
}

func TestGetInfo(t *testing.T) {
	slog.Info("--- TestGetInfo ---")
	const publisherID = "device1"
	const thing0ID = thingIDPrefix + "0"

	// TODO: add GetInfo
	store, readHist, stopFn := newHistoryService()
	defer stopFn()
	_ = readHist
	addBulkHistory(store, 1000, 5, 1000)

	//info := store.Info()
	//t.Logf("Store ID:%s, records:%d", info.Id, info.NrRecords)

	//info := readHistory.Info(ctx)
	//assert.NotEmpty(t, info.Engine)
	//assert.NotEmpty(t, info.Id)
	//t.Logf("ID:%s records:%d", info.Id, info.NrRecords)
}

func TestPubSub(t *testing.T) {
	const agent1ID = "device1"
	const thing0ID = thingIDPrefix + "0"

	slog.Info("--- TestPubSub ---")
	// start the pubsub server
	//svcConfig.Retention = []history.RetentionRule{
	//	{Name: vocab.VocabTemperature},
	//	{Name: vocab.VocabBatteryLevel},
	//}

	svc, readHist, stopFn := newHistoryService()
	defer stopFn()
	_ = svc

	hc1, err := testServer.AddConnectClient(agent1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	defer hc1.Disconnect()
	// publish events
	names := []string{
		vocab.VocabTemperature, vocab.VocabSwitch,
		vocab.VocabSwitch, vocab.VocabBatteryLevel,
		vocab.VocabAlarm, "noname",
		"tttt", vocab.VocabTemperature,
		vocab.VocabSwitch, vocab.VocabTemperature}
	_ = names

	// only valid names should be added
	for i := 0; i < 10; i++ {
		val := strconv.Itoa(i + 1)
		err = hc1.PubEvent(thing0ID, names[i], []byte(val))
		assert.NoError(t, err)
		// make sure timestamp differs
		time.Sleep(time.Millisecond * 3)
	}

	time.Sleep(time.Millisecond * 100)
	// read back
	cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing0ID, "")
	require.NoError(t, err)
	ev, valid, err := cursor.First()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, ev)

	// store

	batched, _, _ := cursor.NextN(10)
	// expect 3 entries total from valid events (9 when retention manager isn't used)
	assert.Equal(t, 9, len(batched))
	releaseFn()
}

func TestManageRetention(t *testing.T) {
	slog.Info("--- TestManageRetention ---")
	const client1ID = "admin"
	const device1ID = "newdevice" // should not match existing test devices
	const thing0ID = thingIDPrefix + "0"
	const event1Name = "event1"
	const event2Name = "notRetainedEvent"

	// setup with some history
	store, readHist, closeFn := newHistoryService()
	defer closeFn()
	addBulkHistory(store, 1000, 5, 1000)

	// connect as an admin user
	hc1, err := testServer.AddConnectClient(client1ID, authapi.ClientTypeUser, authapi.ClientRoleAdmin)
	mngHist := historyclient.NewManageHistoryClient(hc1)

	// should be able to read the current retention rules. Expect the default rules.
	rules1, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Greater(t, 1, len(rules1))

	// Add two retention rules to retain temperature and our test event from device1
	rules1[vocab.VocabTemperature] = append(rules1[vocab.VocabTemperature],
		&historyapi.RetentionRule{Retain: true})
	rules1[event1Name] = append(rules1[event1Name],
		&historyapi.RetentionRule{AgentID: device1ID, Retain: true})
	err = mngHist.SetRetentionRules(rules1)
	require.NoError(t, err)

	// The new retention rule should now exist and accept our custom event
	rules2, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Equal(t, len(rules1), len(rules2))
	rule, err := mngHist.GetRetentionRule(device1ID, thing0ID, event1Name)
	assert.NoError(t, err)
	if assert.NotNil(t, rule) {
		assert.Equal(t, device1ID, rule.AgentID)
		assert.Equal(t, "", rule.ThingID)
		assert.Equal(t, event1Name, rule.Name)
	}

	// connect as device1 and publish two events, one to be retained
	hc2, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.PubEvent(thing0ID, event1Name, []byte("hi)"))
	assert.NoError(t, err)
	err = hc2.PubEvent(thing0ID, event2Name, []byte("hi)"))
	assert.NoError(t, err)
	// give it some time to persist the bucket
	time.Sleep(time.Millisecond * 100)

	// read the history of device 1 and expect the event to be retained
	cursor, releaseFn, err := readHist.GetCursor(device1ID, thing0ID, "")
	require.NoError(t, err)
	histEv1, valid, _ := cursor.First()
	require.True(t, valid, "missing the first event")
	assert.Equal(t, event1Name, histEv1.Name)
	histEv2, valid2, _ := cursor.Next()
	require.False(t, valid2, "second event should not be there")
	_ = histEv2
	releaseFn()

}
