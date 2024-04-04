package valueservice_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/valueservice"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"
)

var testFolder = path.Join(os.TempDir(), "test-valuestore")
var testStoreFile = path.Join(testFolder, "values.json")

const backend = buckets.BackendPebble

var names = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

// generate a random batch of values for testing
func makeValueBatch(valueSvc *valueservice.ValueService,
	nrValues int, thingIDs []string, timespanSec int) (batch []*things.ThingValue) {

	valueBatch := make([]*things.ThingValue, 0, nrValues)
	for j := 0; j < nrValues; j++ {
		thingIndex := rand.Intn(len(thingIDs))
		thingID := thingIDs[thingIndex]
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		randomTime := time.Now().Add(-randomSeconds)

		ev := things.NewThingValue(transports.MessageTypeEvent,
			thingID, names[randomName],
			[]byte(fmt.Sprintf("%2.3f", randomValue)), "sender1",
		)
		ev.CreatedMSec = randomTime.UnixMilli()

		valueSvc.HandleAddValue(ev)
		valueBatch = append(valueBatch, ev)
	}
	return valueBatch
}

func startValueService() (svc *valueservice.ValueService, stopFn func()) {
	var valueStore buckets.IBucketStore
	var cfg = valueservice.NewValueStoreConfig()

	valueStore = bucketstore.NewBucketStore(testFolder, "values", backend)
	err := valueStore.Open()

	if err != nil {
		panic("unable to open value store")
	}
	svc = valueservice.NewThingValueService(&cfg, valueStore)
	err = svc.Start()
	if err != nil {
		err = fmt.Errorf("can't open history bucket store: %w", err)
	}
	return svc, func() {
		svc.Stop()
		valueStore.Close()
	}
}

func TestGetLatest(t *testing.T) {
	t.Log("--- TestGetLatest ---")
	const count = 100
	const agent1ID = "agent1"
	const thing1ID = "thing1" // matches a percentage of the random things

	svc, closeFn := startValueService()
	defer closeFn()

	batch := makeValueBatch(svc, count, []string{thing1ID}, 3600*24*30)
	_ = batch

	t0 := time.Now()
	values := svc.GetProperties(thing1ID, nil)
	assert.NotNil(t, values)
	d0 := time.Now().Sub(t0)

	// 2nd time from cache
	t1 := time.Now()
	values2 := svc.GetProperties(thing1ID, nil)
	assert.NotNil(t, values2)
	d1 := time.Now().Sub(t1)

	assert.Less(t, d1, d0)

	values3 := svc.GetProperties(thing1ID, names)
	_ = values3

	// save and reload props
	svc.Stop()
	err := svc.Start()

	assert.NoError(t, err)
	found := svc.LoadProps(thing1ID)
	assert.False(t, found) // not cached

	//cursor, releaseFn, err := readHist.GetCursor(agent1ID, thing1ID, "")
	//defer releaseFn()

	//t.Logf("Received %d values", len(values))
	//assert.Greater(t, len(values), 0, "Expected multiple properties, got none")
	//// compare the results with the highest value tracked during creation of the test data
	//for _, val := range values {
	//	t.Logf("Result; name '%s'; created: %d", val.Name, val.CreatedMSec)
	//	highest := highestFromAdded[val.Name]
	//	if assert.NotNil(t, highest) {
	//		t.Logf("Expect %s: %v", highest.Name, highest.CreatedMSec)
	//		assert.Equal(t, highest.CreatedMSec, val.CreatedMSec)
	//	}
	//}
	//// getting the Last should get the same result
	//lastItem, valid, _ := cursor.Last()
	//highest := highestFromAdded[lastItem.Name]
	//
	//assert.True(t, valid)
	//assert.Equal(t, lastItem.CreatedMSec, highest.CreatedMSec)
}

func TestAddPropsEvent(t *testing.T) {
	thing1ID := "thing1"
	pev := make(map[string]string)
	pev["temperature"] = "10"
	pev["humidity"] = "33"
	pev["switch"] = "false"
	serProps, _ := json.Marshal(pev)

	svc, closeFn := startValueService()
	defer closeFn()
	tv := things.NewThingValue(transports.MessageTypeEvent,
		thing1ID, transports.EventNameProps, serProps, "sender")
	svc.HandleAddValue(tv)

	values1 := svc.GetProperties(thing1ID, names)
	assert.LessOrEqual(t, len(pev), len(values1))
}

func TestReadPropsFail(t *testing.T) {
	thing1ID := "badthingid"
	svc, closeFn := startValueService()
	defer closeFn()
	values1 := svc.GetProperties(thing1ID, names)
	assert.Empty(t, values1)
}
