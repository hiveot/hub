package service_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
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

var testValueFolder = path.Join(os.TempDir(), "test-values")
var valueStorePath = path.Join(testValueFolder, "values.store")
var valueNames = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

func startLatestStore(clean bool) (
	svc *service.DigiTwinLatestStore,
	stopFn func()) {

	if clean {
		_ = os.Remove(valueStorePath)
	}
	store := kvbtree.NewKVStore(valueStorePath)
	err := store.Open()
	if err != nil {
		panic("unable to open the digital twin bucket store")
	}
	bucket := store.GetBucket("test")
	svc = service.NewDigiTwinLatestStore(bucket)
	err = svc.Start()
	if err != nil {
		panic("unable to start the latest store service")
	}

	return svc, func() {
		svc.Stop()
		_ = store.Close()
	}
}

// generate a random batch of values for testing
func createValueBatch(svc *service.DigiTwinLatestStore,
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
			fmt.Sprintf("%2.3f", randomValue),
			"sender1",
		)
		ev.CreatedMSec = randomTime.UnixMilli()

		svc.StoreMessage(ev)
		valueBatch = append(valueBatch, ev)
	}
	return valueBatch
}

func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testValueFolder)
	err = os.MkdirAll(testValueFolder, 0700)

	if err != nil {
		panic(err)
	}

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, stopFunc := startLatestStore(true)

	createValueBatch(svc, 100, thingIDs, 100)

	valList1, err := svc.ReadLatest(vocab.MessageTypeEvent, thingIDs[1], nil, "")
	//valList1, err := cl.ReadProperties(thingIDs[1], nil)
	assert.NoError(t, err)
	assert.True(t, len(valList1) > 1)

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, stopFunc = startLatestStore(false)
	defer stopFunc()
	//tdList2, err := directory.ReadThings(mt, 0, 10)
	//assert.Equal(t, len(thingIDs), len(tdList2))

	valList2a, err := svc.ReadLatest(vocab.MessageTypeEvent, thingIDs[1], nil, "")
	assert.NoError(t, err)
	assert.Equal(t, len(valList1), len(valList2a))
	valList2b, err := svc.ReadLatest(vocab.MessageTypeEvent, thingIDs[1], nil, "")
	_ = valList2b
	assert.NoError(t, err)
	valList2c, err := svc.ReadLatest(vocab.MessageTypeAction, thingIDs[1], nil, "")
	_ = valList2c
	assert.NoError(t, err)
}

func TestGetEvents(t *testing.T) {
	const count = 100
	const agent1ID = "agent1"
	const thing1ID = "thing1" // matches a percentage of the random things

	svc, closeFn := startLatestStore(true)
	defer closeFn()

	batch := createValueBatch(svc, count, []string{thing1ID}, 3600*24*30)
	_ = batch

	t0 := time.Now()

	values, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, nil, "")
	require.NoError(t, err)
	require.NotNil(t, values)
	d0 := time.Now().Sub(t0)

	// 2nd time from cache
	t1 := time.Now()
	values2, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, nil, "")
	require.NoError(t, err)
	require.NotNil(t, values2)
	d1 := time.Now().Sub(t1)

	assert.Less(t, d1, d0)

	values3, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, valueNames, "")
	require.NoError(t, err)
	require.NotNil(t, values3)
	_ = values3

	// save and reload props
	closeFn()
	//svc.Stop()
	svc, closeFn = startLatestStore(false)
	//err = svc.Start()
	assert.NoError(t, err)

	// LoadLatest should load and not find a cached value
	found := svc.LoadLatest(thing1ID)
	assert.False(t, found) // cache reload
	// but the value should still be there
	values4, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, valueNames, "")
	require.NoError(t, err)
	require.NotNil(t, values4)
}

func TestAddPropsEvent(t *testing.T) {
	thing1ID := "thing1"
	pev := make(map[string]string)
	pev["temperature"] = "10"
	pev["humidity"] = "33"
	pev["switch"] = "false"
	serProps, _ := json.Marshal(pev)

	svc, closeFn := startLatestStore(true)
	defer closeFn()

	msg := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, string(serProps), "sender")
	svc.StoreMessage(msg)

	values1, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Equal(t, len(pev), len(values1))
}

func TestAddPropsFail(t *testing.T) {
	thing1ID := "badthingid"
	svc, closeFn := startLatestStore(true)
	_ = svc
	defer closeFn()
	values1, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Empty(t, values1)
}

func TestAddBadProps(t *testing.T) {
	thing1ID := "thing1"
	badProps := []string{"bad1", "bad2"}
	serProps, _ := json.Marshal(badProps)

	svc, closeFn := startLatestStore(true)
	defer closeFn()
	msg := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProperties, string(serProps), "sender")
	svc.StoreMessage(msg)

	//// action is ignored
	//tv.MessageType = vocab.MessageTypeAction
	//_, err = svc.StoreEvent(tv)
	//assert.NoError(t, err)
	//
	values1, err := svc.ReadLatest(vocab.MessageTypeEvent, thing1ID, valueNames, "")
	require.NoError(t, err)
	assert.Equal(t, 0, len(values1))
}
