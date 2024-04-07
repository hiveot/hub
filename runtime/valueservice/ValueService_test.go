package valueservice_test

import (
	"encoding/json"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
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
			thingID, names[randomName],
			[]byte(fmt.Sprintf("%2.3f", randomValue)), "sender1",
		)
		ev.CreatedMSec = randomTime.UnixMilli()

		_, _ = valueSvc.HandleEvent(ev)
		valueBatch = append(valueBatch, ev)
	}
	return valueBatch
}

// start the value service
// optionally clean the existing store
func startValueService(clean bool) (svc *valueservice.ValueService, stopFn func()) {
	var valueStore buckets.IBucketStore
	var cfg = valueservice.NewValueStoreConfig()

	if clean {
		_ = os.RemoveAll(testFolder)
		_ = os.Mkdir(testFolder, 0700)
	}
	valueStore, _ = bucketstore.NewBucketStore(testFolder, "values", backend)
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

	svc, closeFn := startValueService(true)
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
}

func TestAddPropsEvent(t *testing.T) {
	thing1ID := "thing1"
	pev := make(map[string]string)
	pev["temperature"] = "10"
	pev["humidity"] = "33"
	pev["switch"] = "false"
	serProps, _ := json.Marshal(pev)

	svc, closeFn := startValueService(true)
	defer closeFn()
	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProps, serProps, "sender")
	_, _ = svc.HandleEvent(tv)

	values1 := svc.GetProperties(thing1ID, names)
	assert.Equal(t, len(pev), len(values1))
}

func TestAddBadProps(t *testing.T) {
	thing1ID := "thing1"
	badProps := []string{"bad1", "bad2"}
	serProps, _ := json.Marshal(badProps)

	svc, closeFn := startValueService(true)
	defer closeFn()
	tv := things.NewThingMessage(vocab.MessageTypeEvent,
		thing1ID, vocab.EventTypeProps, serProps, "sender")
	_, err := svc.HandleEvent(tv)
	assert.Error(t, err)

	// action is ignored
	tv.MessageType = vocab.MessageTypeAction
	_, err = svc.HandleEvent(tv)
	assert.NoError(t, err)

	values1 := svc.GetProperties(thing1ID, names)
	assert.Equal(t, 0, len(values1))
}

func TestReadPropsFail(t *testing.T) {
	thing1ID := "badthingid"
	svc, closeFn := startValueService(true)
	defer closeFn()
	values1 := svc.GetProperties(thing1ID, names)
	assert.Empty(t, values1)
}
