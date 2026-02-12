package history_test

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/messaging"
	"github.com/hiveot/hub/lib/testenv"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/services/history/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

const thingIDPrefix = "things-"

// recommended store for history is Pebble
// const historyStoreBackend = buckets.BackendPebble
const historyStoreBackend = buckets.BackendKVBTree

const testClientID = "operator1"

// the following are set by the testmain
var ts *testenv.TestServer

var names = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

// Create a new store, delete if it already exists
func startHistoryService(clean bool) (
	svc *service.HistoryService,
	r *historyclient.ReadHistoryClient,
	stopFn func()) {

	ts = testenv.StartTestServer(clean)

	histStore, err := bucketstore.NewBucketStore(
		ts.TestDir, "hist", historyStoreBackend)
	if err == nil {
		err = histStore.Open()
	}
	if err != nil {
		panic("can't open history bucket store")
	}

	// the service needs an agent connection
	agentConn, _ := ts.AddConnectService(historyapi.AgentID)
	svc = service.NewHistoryService(histStore)

	err = svc.Start(agentConn)
	if err != nil {
		panic("Failed starting the state service: " + err.Error())
	}

	// create an end user client for testing
	co1, _, _ := ts.AddConnectConsumer(testClientID, authz.ClientRoleOperator)
	histCl := historyclient.NewReadHistoryClient(co1)

	return svc, histCl, func() {
		co1.Disconnect()
		svc.Stop()
		_ = histStore.Close()
		agentConn.Disconnect()
		ts.Stop()
		// give it some time to shut down before the next test
		time.Sleep(time.Millisecond)
	}
}

//func stopStore(store client.IHistory) error {
//	return store.(*mongohs.MongoHistoryServer).Stop()
//}

// generate a random batch of property and event values for testing
// timespanSec is the range of timestamps up until now
func makeValueBatch(agentID string, nrValues, nrThings, timespanSec int) (
	batch []messaging.ThingValue, highest map[string]messaging.ThingValue) {

	highest = make(map[string]messaging.ThingValue)
	valueBatch := make([]messaging.ThingValue, 0, nrValues)
	for j := 0; j < nrValues; j++ {
		randomID := rand.Intn(nrThings)
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		randomTime := time.Now().Add(-randomSeconds)
		//
		thingID := thingIDPrefix + strconv.Itoa(randomID)
		dThingID := td.MakeDigiTwinThingID(agentID, thingID)

		randomMsgType := rand.Intn(2)
		affType := messaging.AffordanceTypeEvent
		if randomMsgType == 1 {
			affType = messaging.AffordanceTypeProperty
		}

		tv := messaging.ThingValue{
			//ID:             fmt.Sprintf("%d", randomID),
			Name:           names[randomName],
			Data:           fmt.Sprintf("%2.3f", randomValue),
			ThingID:        dThingID,
			Timestamp:      utils.FormatUTCMilli(randomTime),
			AffordanceType: affType,
		}

		// track the actual most recent event for the name for things 3
		if randomID == 0 {
			if _, exists := highest[tv.Name]; !exists ||
				highest[tv.Name].Timestamp < tv.Timestamp {
				highest[tv.Name] = tv
			}
		}
		valueBatch = append(valueBatch, tv)
	}
	return valueBatch, highest
}

// add some history to the store. This bypasses the check for thingID to exist.
func addBulkHistory(svc *service.HistoryService, agentID string, count int, nrThings int,
	timespanSec int) (highest map[string]messaging.ThingValue) {

	var batchSize = 1000
	if batchSize > count {
		batchSize = count
	}

	evBatch, highest := makeValueBatch(agentID, count, nrThings, timespanSec)
	addHist := svc.GetAddHistory()

	// use add multiple in 100's
	for i := 0; i < count/batchSize; i++ {
		// no thingID constraint allows adding events from any things
		start := batchSize * i
		end := batchSize * (i + 1)
		for j := start; j < end; j++ {
			err := addHist.AddValue(agentID, evBatch[j])
			if err != nil {
				slog.Error("Problem adding events.", "err", err)
			}
		}
	}
	return highest
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

// Test creating and deleting the history database
// This requires a local unsecured MongoDB instance
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	store, readHist, stopFn := startHistoryService(true)
	defer stopFn()
	_ = store
	_ = readHist
}

func TestAddGetEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const id1 = "thing1"
	const id2 = "thing2"
	const agent1ID = "agent1"
	const thing1ID = id1
	const thing2ID = id2
	const evTemperature = "temperature"
	const evHumidity = "humidity"

	// add history with two specific things to test against
	svc, readHist, stopFn := startHistoryService(true)
	// do not defer cancel as it will be closed and reopened in the test
	fivemago := time.Now().Add(-time.Minute * 5)
	fiftyfivemago := time.Now().Add(-time.Minute * 55)
	addBulkHistory(svc, agent1ID, 20, 3, 3600)

	// add thing1 temperature from 5 minutes ago
	addHist := svc.GetAddHistory()
	dThing1ID := td.MakeDigiTwinThingID(agent1ID, thing1ID)
	ev1_1 := &messaging.NotificationMessage{
		Operation: wot.OpSubscribeEvent,
		SenderID:  agent1ID, ThingID: dThing1ID, Name: evTemperature,
		Value: "12.5", Timestamp: utils.FormatUTCMilli(fivemago),
	}
	err := addHist.AddMessage(ev1_1)
	assert.NoError(t, err)
	// add thing1 humidity from 55 minutes ago
	ev1_2 := &messaging.NotificationMessage{
		Operation: vocab.OpSubscribeEvent,
		SenderID:  agent1ID, ThingID: dThing1ID, Name: evHumidity,
		Value: "70", Timestamp: utils.FormatUTCMilli(fiftyfivemago),
	}
	err = addHist.AddMessage(ev1_2)
	assert.NoError(t, err)

	// add thing2 humidity from 5 minutes ago
	dThing2ID := td.MakeDigiTwinThingID(agent1ID, thing2ID)
	ev2_1 := &messaging.NotificationMessage{
		Operation: vocab.OpSubscribeEvent,
		SenderID:  agent1ID, ThingID: dThing2ID, Name: evHumidity,
		Value: "50", Timestamp: utils.FormatUTCMilli(fivemago),
	}
	err = addHist.AddMessage(ev2_1)
	assert.NoError(t, err)

	// add thing2 temperature from 55 minutes ago
	ev2_2 := &messaging.NotificationMessage{
		Operation: vocab.OpSubscribeEvent,
		SenderID:  agent1ID, ThingID: dThing2ID, Name: evTemperature,
		Value: "17.5", Timestamp: utils.FormatUTCMilli(fiftyfivemago),
	}
	err = addHist.AddMessage(ev2_2)
	assert.NoError(t, err)

	// Test 1: get events of thing1 older than 300 minutes ago - expect 1 humidity from 55 minutes ago
	cursor1, c1Release, err := readHist.GetCursor(dThing1ID, "")
	require.NoError(t, err)

	// seek must return the things humidity added 55 minutes ago, not 5 minutes ago
	timeAfter := time.Now().Add(-time.Minute * 300)
	tv1, valid, err := cursor1.Seek(timeAfter)
	if assert.NoError(t, err) && assert.True(t, valid) {
		assert.Equal(t, dThing1ID, tv1.ThingID)
		assert.Equal(t, evHumidity, tv1.Name)
		// next finds the temperature from 5 minutes ago
		tv2, valid, err := cursor1.Next()
		assert.NoError(t, err)
		if assert.True(t, valid) {
			assert.Equal(t, evTemperature, tv2.Name)
		}
	}

	// Test 2: get events of things 1 newer than 30 minutes ago - expect 1 temperature
	timeAfter = time.Now().Add(-time.Minute * 30)

	// do we need to get a new cursor?
	//readHistory = svc.CapReadHistory()
	tv3, valid, _ := cursor1.Seek(timeAfter)
	if assert.True(t, valid) {
		assert.Equal(t, dThing1ID, tv3.ThingID)  // must match the filtered id1
		assert.Equal(t, evTemperature, tv3.Name) // must match evTemperature from 5 minutes ago
		assert.Equal(t, utils.FormatUTCMilli(fivemago), tv3.Timestamp)
	}
	c1Release()
	// Stop the service before phase 2
	stopFn()

	// PHASE 2: after closing and reopening the svc the event should still be there
	svc, readHist, stopFn = startHistoryService(false)
	defer stopFn()

	// Test 3: get first temperature of things 2 - expect 1 result
	time.Sleep(time.Second)
	cursor2, releaseFn, err := readHist.GetCursor(dThing2ID, "")
	require.NoError(t, err)
	tv4, valid, err := cursor2.First()
	require.NoError(t, err)
	require.True(t, valid)
	assert.Equal(t, evTemperature, tv4.Name)
	releaseFn()
}

func TestAddProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	//const clientID = "device0"
	const thing1ID = thingIDPrefix + "0" // matches a percentage of the random things
	const agent1 = "device1"
	const temp1 = 55
	const battTemp = 50

	svc, readHist, closeFn := startHistoryService(true)
	defer closeFn()

	dThing1ID := td.MakeDigiTwinThingID(agent1, thing1ID)
	action1 := &messaging.NotificationMessage{
		SenderID:  agent1,
		ThingID:   dThing1ID,
		Name:      vocab.ActionSwitchOnOff,
		Value:     "on",
		Operation: vocab.OpInvokeAction,
	}
	event1 := &messaging.NotificationMessage{
		SenderID:  agent1,
		ThingID:   dThing1ID,
		Name:      vocab.PropEnvTemperature,
		Value:     temp1,
		Operation: vocab.OpSubscribeEvent,
	}
	badEvent1 := &messaging.NotificationMessage{
		SenderID:  agent1,
		ThingID:   dThing1ID,
		Name:      "", // missing name
		Operation: vocab.OpSubscribeEvent,
	}
	// dThing1ID identifies the publisher so not an error
	//badEvent2 := &transports.IConsumer{
	//	SenderID:  "", // missing publisher
	//	ThingID:   dThing1ID,
	//	Name:      "name",
	//	Operation: vocab.OpSubscribeEvent,
	//}
	badEvent3 := &messaging.NotificationMessage{
		SenderID:  agent1,
		ThingID:   dThing1ID,
		Name:      "baddate",
		Timestamp: "-1",
		Operation: vocab.OpSubscribeEvent,
	}
	badEvent4 := &messaging.NotificationMessage{
		SenderID: agent1,
		ThingID:  "", // missing ID
		Name:     "temperature",
	}
	propsList := make(map[string]interface{})
	propsList[vocab.PropDeviceBattery] = battTemp
	propsList[vocab.PropEnvCpuload] = 30
	propsList[vocab.PropSwitchOnOff] = "off"
	props1 := &messaging.NotificationMessage{
		SenderID:  agent1,
		ThingID:   dThing1ID,
		Name:      "", // property list
		Value:     propsList,
		Operation: wot.OpObserveAllProperties,
	}

	// in total add 5 properties
	addHist := svc.GetAddHistory()
	err := addHist.AddMessage(action1)
	assert.NoError(t, err)
	err = addHist.AddMessage(event1)
	assert.NoError(t, err)
	err = addHist.AddMessage(props1)
	assert.NoError(t, err)

	// and some bad values
	err = addHist.AddMessage(badEvent1)
	assert.Error(t, err)
	//err = addHist.AddMessage(badEvent2)
	//assert.Error(t, err)
	err = addHist.AddMessage(badEvent3) // bad date is recovered
	assert.NoError(t, err)
	err = addHist.AddMessage(badEvent4)
	assert.Error(t, err)
	err = addHist.AddMessage(badEvent1)
	assert.Error(t, err)

	c, releaseFn, err := readHist.GetCursor(dThing1ID, "")
	defer releaseFn()
	require.NoError(t, err)
	msg, valid, err := c.First()
	require.True(t, valid)
	assert.NotEmpty(t, msg)
	hasProps := false
	for valid && err == nil {
		if msg.AffordanceType == messaging.AffordanceTypeProperty {
			hasProps = true
			require.NotEmpty(t, msg.Name)
			require.NotEmpty(t, msg.Data)
			if msg.Name == vocab.PropDeviceBattery {
				assert.Equal(t, float64(battTemp), msg.Data)
			}
			//props := make(map[string]interface{})
			//err = utils.DecodeAsObject(msg.Data, &props)
			//require.NoError(t, err)
		} else if msg.Name == vocab.PropEnvTemperature {
			dataInt := utils.DecodeAsInt(msg.Data)
			require.Equal(t, temp1, dataInt)
		}
		msg, valid, err = c.Next()
	}
	require.NoError(t, err)
	require.True(t, hasProps)
}

func TestGetInfo(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	//const thing0ID = thingIDPrefix + "0"
	//var dThing0ID = things.MakeDigiTwinThingID(agentID, thing0ID)

	store, readHist, stopFn := startHistoryService(true)
	defer stopFn()
	_ = readHist
	addBulkHistory(store, agentID, 1000, 5, 1000)

	// TODO: add GetInfo for store statistics
	//info := store.Info()
	//t.Logf("Store ID:%s, records:%d", info.Id, info.NrRecords)

	//info := readHistory.Info(ctx)
	//assert.NotEmpty(t, info.Engine)
	//assert.NotEmpty(t, info.Id)
	//t.Logf("ID:%s records:%d", info.Id, info.NrRecords)
}

func TestPrevNext(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things
	var dThing0ID = td.MakeDigiTwinThingID(agentID, thing0ID)

	store, readHist, closeFn := startHistoryService(true)
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(store, agentID, count, 1, 3600*24*30)

	cursor, releaseFn, _ := readHist.GetCursor(dThing0ID, "")
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
	items2to11, itemsRemaining, err := cursor.NextN(10, "")
	require.NoError(t, err)
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))

	// go backwards
	item10to1, itemsRemaining, err := cursor.PrevN(10, "")
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := cursor.Prev()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item0.Timestamp, item0b.Timestamp)

	// can't skip before the beginning of time
	iteminv, valid, err := cursor.Prev()
	require.NoError(t, err)
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	timeStamp, _ := dateparse.ParseAny(item11.Timestamp)
	item11b, valid, err := cursor.Seek(timeStamp)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item11.Name, item11b.Name)
}

// filter on property name
func TestPrevNextFiltered(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things
	var dThing0ID = td.MakeDigiTwinThingID(agentID, thing0ID)

	svc, readHist, closeFn := startHistoryService(true)
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(svc, agentID, count, 1, 3600*24*30)
	propName := names[2] // names was used to generate the history

	// A cursor with a filter on propName should only return results of propName
	cursor, releaseFn, err := readHist.GetCursor(dThing0ID, propName)
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
	require.Nil(t, err)
	assert.Equal(t, propName, item1.Name)
	items2to11, itemsRemaining, err := cursor.NextN(10, "")
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))
	assert.Equal(t, propName, items2to11[9].Name)

	// go backwards
	item10to1, itemsRemaining, err := cursor.PrevN(10, "")
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := cursor.Prev()
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, item0.Timestamp, item0b.Timestamp)
	assert.Equal(t, propName, item0b.Name)

	// can't skip before the beginning of time
	iteminv, valid, err := cursor.Prev()
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	timeStamp, _ := dateparse.ParseAny(item11.Timestamp)
	item11b, valid, err := cursor.Seek(timeStamp)
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, item11.Name, item11b.Name)

	// last item should be of the name
	lastItem, valid, err := cursor.Last()
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, propName, lastItem.Name)

	cursor.Release()
}

func TestNextPrevUntil(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things
	var dThing0ID = td.MakeDigiTwinThingID(agentID, thing0ID)

	store, readHist, closeFn := startHistoryService(true)
	defer closeFn()

	// 1 sensor -> 1000/24 hours is approx 41/hour
	_ = addBulkHistory(store, agentID, count, 1, 3600*24)

	cursor, releaseFn, _ := readHist.GetCursor(dThing0ID, "")
	defer releaseFn()
	assert.NotNil(t, cursor)

	// start 20 hours ago
	startTime := time.Now().Add(-20 * time.Hour)
	item0, valid, err := cursor.Seek(startTime)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item0)

	// read an hour's worth. Expect around 41 results
	endTime := startTime.Add(time.Hour).Format(time.RFC3339)
	// note, batch1 doesn't have item0
	batch, itemsRemaining, err := cursor.NextN(100, endTime)
	require.NoError(t, err)
	assert.False(t, itemsRemaining)
	assert.True(t, len(batch) > 20)
	assert.True(t, len(batch) < 60)

	// read backwards again. note batch2 ends with item0
	batch2, itemsRemaining, err := cursor.PrevN(100, startTime.Format(time.RFC3339))
	require.NoError(t, err)
	assert.False(t, itemsRemaining)
	assert.Equal(t, len(batch), len(batch2))
}

func TestReadHistory(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "device1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things
	var dThing0ID = td.MakeDigiTwinThingID(agentID, thing0ID)

	store, readHist, closeFn := startHistoryService(true)
	defer closeFn()

	// 1 sensors -> 1000/24 hours is approx 41/hour
	_ = addBulkHistory(store, agentID, count, 1, 3600*24)

	// start 20 hours ago and read an hour's worth
	startTime := time.Now().Add(-20 * time.Hour)
	duration := time.Hour
	items, remaining, err := readHist.ReadHistory(dThing0ID, "", startTime, duration, 60)
	require.NoError(t, err)
	assert.False(t, remaining)
	assert.NotEmpty(t, items)
	assert.True(t, len(items) > 20)

	// start 19 hours ago and read back in time
	startTime = time.Now().Add(-19 * time.Hour)
	duration = -1 * time.Hour
	items, remaining, err = readHist.ReadHistory(dThing0ID, "", startTime, duration, 60)
	require.NoError(t, err)
	assert.False(t, remaining)
	assert.NotEmpty(t, items)
	assert.True(t, len(items) > 20)

}

func TestPubEvents(t *testing.T) {
	const agent1ID = "device1"

	t.Logf("---%s---\n", t.Name())

	_, readHist, stopFn := startHistoryService(true)
	defer stopFn()
	ag1, _ := ts.AddConnectService(agent1ID)
	defer ag1.Disconnect()

	// Add the thing who is publishing events
	td1 := ts.CreateTestTD(0)
	ts.AddTD(agent1ID, td1)
	thing0ID := td1.ID
	dThing0ID := td.MakeDigiTwinThingID(agent1ID, thing0ID)

	// publish events
	names := []string{
		vocab.PropEnvTemperature, vocab.PropSwitchOnOff,
		vocab.PropSwitchOnOff, vocab.PropDeviceBattery,
		vocab.PropAlarmStatus, "noname",
		"tttt", vocab.PropEnvTemperature,
		vocab.PropSwitchOnOff, vocab.PropEnvTemperature}
	_ = names

	// only valid names should be added
	for i := 0; i < 10; i++ {
		val := strconv.Itoa(i + 1)
		// events are published by the agent using their native thingID
		name := names[i]
		err := ag1.PubEvent(thing0ID, name, val)
		//err := ag1.PubEvent(f1, thing0ID, name, val, "")
		assert.NoError(t, err)
		// make sure timestamp differs
		time.Sleep(time.Millisecond * 3)
	}

	time.Sleep(time.Millisecond * 1000)
	// read back
	// consumers read events see the digital twin representation
	cursor, releaseFn, err := readHist.GetCursor(dThing0ID, "")
	require.NoError(t, err)
	ev, valid, err := cursor.First()
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, ev)

	// store

	batched, _, _ := cursor.NextN(10, "")
	// expect 3 entries total from valid events (9 when retention manager isn't used)
	assert.Equal(t, 9, len(batched))
	releaseFn()
}

func TestManageRetention(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const client1ID = "admin"
	const agentID = "agent1" // should not match existing test devices
	const event1Name = "event1"
	const event2Name = "notRetainedEvent"

	// setup with some history
	store, readHist, closeFn := startHistoryService(true)
	defer closeFn()
	addBulkHistory(store, agentID, 1000, 5, 1000)

	// make sure the TD whose retention rules are added exist
	td0 := ts.CreateTestTD(0)
	ts.AddTD(agentID, td0)
	dThing0ID := td.MakeDigiTwinThingID(agentID, td0.ID)

	// connect as an admin user
	co1, _, _ := ts.AddConnectConsumer(client1ID, authz.ClientRoleAdmin)
	mngHist := historyclient.NewManageHistoryClient(co1)

	// should be able to read the current retention rules. Expect the default rules.
	rules1, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Greater(t, 1, len(rules1))

	// Add two retention rules to retain temperature and our test event from device1
	rules1[vocab.PropEnvTemperature] = append(rules1[vocab.PropEnvTemperature],
		&historyapi.RetentionRule{Retain: true})
	rules1[event1Name] = append(rules1[event1Name],
		&historyapi.RetentionRule{Retain: true})
	err = mngHist.SetRetentionRules(rules1)
	require.NoError(t, err)

	// The new retention rule should now exist and accept our custom event
	rules2, err := mngHist.GetRetentionRules()
	require.NoError(t, err)
	assert.Equal(t, len(rules1), len(rules2))
	rule, err := mngHist.GetRetentionRule(dThing0ID, event1Name)
	assert.NoError(t, err)
	if assert.NotNil(t, rule) {
		assert.Equal(t, "", rule.ThingID)
		assert.Equal(t, event1Name, rule.Name)
	}

	// connect as agent-1 and publish two events for thing0, one to be retained
	ag1, _, _ := ts.AddConnectAgent(agentID)
	require.NoError(t, err)
	defer ag1.Disconnect()
	err = ag1.PubEvent(td0.ID, event1Name, "event one")
	assert.NoError(t, err)
	err = ag1.PubEvent(td0.ID, event2Name, "event two")
	assert.NoError(t, err)
	// give it some time to persist the bucket
	time.Sleep(time.Millisecond * 100)

	// read the history of device 1 and expect the event to be retained
	cursor, releaseFn, err := readHist.GetCursor(dThing0ID, "")
	require.NoError(t, err)
	histEv1, valid, _ := cursor.First()
	require.True(t, valid, "missing the first event")
	assert.Equal(t, event1Name, histEv1.Name)
	histEv2, valid2, _ := cursor.Next()
	assert.False(t, valid2, "second event should not be there")
	_ = histEv2
	releaseFn()

}
