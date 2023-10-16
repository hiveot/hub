package history_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/core/history/config"
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

const core = "mqtt"
const historyStoreBackend = buckets.BackendPebble
const serviceID = history.ServiceName
const testClientID = "operator1"

// the following are set by the testmain
var testServer *testenv.TestServer

//var svcConfig = config.HistoryStoreConfig{
//	DatabaseType:    "mongodb",
//	DatabaseName:    "test",
//	DatabaseURL:     config.DefaultDBURL,
//	LoginName:       "",
//	Password:        "",
//	ClientCertificate: "",
//}

var names = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

// var testItems = make(map[string]thing.ThingValue)
//var testHighestName = make(map[string]thing.ThingValue)

// Create a new store, delete if it already exists
func newHistoryService() (
	store buckets.IBucketStore, r *historyclient.ReadHistoryClient, stopFn func()) {

	svcConfig := config.NewHistoryConfig(testFolder)

	// create a new empty store to use
	_ = os.RemoveAll(svcConfig.StoreDirectory)
	histStore := bucketstore.NewBucketStore(testFolder, serviceID, historyStoreBackend)
	err := histStore.Open()
	if err != nil {
		panic("can't open history bucket store")
	}

	// the service needs a server connection
	hc, err := testServer.AddConnectClient(history.ServiceName, auth.ClientTypeService, auth.ClientRoleService)
	svc := service.NewHistoryService(hc, histStore)
	if err == nil {
		err = svc.Start()
	}
	if err != nil {
		panic("Failed starting the state service: " + err.Error())
	}

	// create an end user client for testing
	hc2, err := testServer.AddConnectClient(testClientID, auth.ClientTypeUser, auth.ClientRoleOperator)
	histCl := historyclient.NewReadHistoryClient(hc2)

	return histStore, histCl, func() {
		hc2.Disconnect()
		_ = svc.Stop()
		_ = histStore.Close()
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
func addBulkHistory(histStore buckets.IBucketStore, count int, nrThings int, timespanSec int) (highest map[string]*thing.ThingValue) {

	const publisherID = "device1"
	var batchSize = 1000
	if batchSize > count {
		batchSize = count
	}

	addHist := service.NewAddHistory(histStore, nil, nil)
	evBatch, highest := makeValueBatch(publisherID, count, nrThings, timespanSec)

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
	testServer, _ = testenv.StartTestServer(core)

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
	histStore, readHist, cancelFn := newHistoryService()
	fivemago := time.Now().Add(-time.Minute * 5)
	fiftyfivemago := time.Now().Add(-time.Minute * 55)
	addBulkHistory(histStore, 20, 3, 3600)
	addHist := service.NewAddHistory(histStore, nil, nil)

	// add thing1 temperature from 5 minutes ago
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
	cursor1, cRelease, err := readHist.GetCursor(agent1ID, thing1ID, "")
	defer cRelease()
	// seek must return the thing humidity added 55 minutes ago, not 5 minutes ago
	timeAfter := time.Now().Add(-time.Minute * 300).UnixMilli()
	tv1, valid, err := cursor1.Seek(timeAfter)
	if assert.NoError(t, err) && assert.True(t, valid) {
		assert.Equal(t, thing1ID, tv1.ThingID)
		assert.Equal(t, evHumidity, tv1.Name)
		// next finds the temperature from 5 minutes ago
		tv2, valid, err := cursor1.Next()
		assert.NoError(t, err)
		assert.True(t, valid)
		assert.Equal(t, evTemperature, tv2.Name)
	}

	// Test 2: get events of thing 1 newer than 30 minutes ago - expect 1 temperature
	timeAfter = time.Now().Add(-time.Minute * 30).UnixMilli()

	// do we need to get a new cursor?
	//readHistory = svc.CapReadHistory()
	res2, valid, _ := cursor1.Seek(timeAfter)
	if assert.True(t, valid) {
		assert.Equal(t, thing1ID, res2.ThingID)   // must match the filtered id1
		assert.Equal(t, evTemperature, res2.Name) // must match evTemperature from 5 minutes ago
		assert.Equal(t, fivemago.UnixMilli(), res2.CreatedMSec)
	}
	cancelFn()

	// PHASE 2: after closing and reopening the svc the event should still be there
	store2, readHist2, cancelFn2 := newHistoryService()
	defer cancelFn2()
	require.NotNil(t, store2)

	// Test 3: get first temperature of thing 2 - expect 1 result
	cursor2, releaseFn, _ := readHist2.GetCursor(agent1ID, thing2ID, "")
	res3, valid, _ := cursor2.First()
	require.True(t, valid)
	assert.Equal(t, evTemperature, res3.Name)
	releaseFn()
}

func TestAddPropertiesEvent(t *testing.T) {
	t.Log("--- TestAddPropertiesEvent ---")
	//const clientID = "device0"
	const thing1ID = thingIDPrefix + "0" // matches a percentage of the random things
	const agent1 = "device1"
	const temp1 = "55"

	store, readHist, closeFn := newHistoryService()
	// do not defer closeFn as a mid-test restart will take place

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
	addHist := service.NewAddHistory(store, nil, nil)
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

	// restart
	closeFn()

	store2, readHist2, closeFn2 := newHistoryService()
	defer closeFn2()
	require.NotNil(t, store2)

	// after closing and reopen the store the properties should still be there
	props, err = readHist2.GetLatest(agent1, thing1ID, []string{vocab.VocabTemperature, vocab.VocabSwitch})
	assert.Equal(t, 2, len(props))
	assert.Equal(t, props[0].Name, vocab.VocabTemperature)
	assert.Equal(t, props[0].Data, []byte(temp1))
	assert.Equal(t, props[1].Name, vocab.VocabSwitch)
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

	store, readHist, closeFn := newHistoryService()
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	// TODO: use different timezones
	_ = addBulkHistory(store, count, 1, 3600*24*30)
	propName := names[2] // names used to generate the history

	values, err := readHist.GetLatest(agent1ID, thing0ID, []string{propName})
	require.NoError(t, err)
	cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing0ID, propName)
	defer releaseFn()

	assert.NotNil(t, values)
	assert.NotNil(t, cursor)

	// go forward
	item0, valid, err := cursor.First()
	assert.True(t, valid)
	assert.Equal(t, propName, item0.Name)
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

	_, readHist, stopFn := newHistoryService()
	defer stopFn()

	hc1, err := testServer.AddConnectClient(agent1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)

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
		err = hc1.PubEvent(thing0ID, names[i], []byte("0.3"))
		assert.NoError(t, err)
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
	batched, _, _ := cursor.NextN(10)
	// expect 4 entries total from valid events
	assert.Equal(t, 3, len(batched))
	releaseFn()
}

func TestManageRetention(t *testing.T) {
	slog.Info("--- TestManageRetention ---")
	const agent1ID = "admin"
	const thing0ID = thingIDPrefix + "0"

	store, readHist, closeFn := newHistoryService()
	defer closeFn()
	addBulkHistory(store, 1000, 5, 1000)

	//info := store.Info(ctx)
	//t.Logf("Store ID:%s, records:%d", info.Id, info.NrRecords)
	hc1, err := testServer.AddConnectClient(agent1ID, auth.ClientTypeUser, auth.ClientRoleAdmin)
	mngHist := historyclient.NewManageHistoryClient(hc1)

	// verify the default retention list
	retList, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Greater(t, 1, len(retList))

	// add a couple of retention
	newRet := &history.RetentionRule{Name: vocab.VocabTemperature}
	err = mngHist.SetRetentionRule(newRet)
	newRet = &history.RetentionRule{
		Name: "blob1", Publishers: []string{agent1ID}}
	err = mngHist.SetRetentionRule(newRet)
	require.NoError(t, err)

	// The new retention should now exist
	retList2, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Equal(t, len(retList)+2, len(retList2))
	ret3, err := mngHist.GetRetentionRule("blob1")
	require.NoError(t, err)
	assert.Equal(t, "blob1", ret3.Name)
	valid, err := mngHist.CheckRetention(&thing.ThingValue{
		AgentID: agent1ID,
		ThingID: thing0ID,
		Name:    "blob1",
	})
	assert.NoError(t, err)
	assert.True(t, valid)

	// events of blob1 should be accepted now
	hc2, err := testServer.AddConnectClient(agent1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)
	require.NoError(t, err)

	err = hc2.PubEvent(thing0ID, "blob1", []byte("hi)"))
	assert.NoError(t, err)
	//
	cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing0ID, "blob1")
	require.NoError(t, err)
	histEv, valid, _ := cursor.First()
	assert.True(t, valid)
	assert.Equal(t, "blob1", histEv.Name)
	releaseFn()
	//
	err = mngHist.RemoveRetentionRule("blob1")
	assert.NoError(t, err)
}
