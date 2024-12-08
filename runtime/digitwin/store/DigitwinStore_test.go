package store_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/wot/td"
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
	svc *store.DigitwinStore,
	stopFn func()) {

	if clean {
		_ = os.Remove(valueStorePath)
	}
	kvstore := kvbtree.NewKVStore(valueStorePath)
	err := kvstore.Open()
	if err != nil {
		panic("unable to open the digital twin bucket store")
	}
	svc, err = store.OpenDigitwinStore(kvstore, false)
	if err != nil {
		panic("unable to start the latest store service")
	}

	return svc, func() {
		svc.Close()
		_ = kvstore.Close()
	}
}

// generate a random batch of values for testing
func addValues(svc *store.DigitwinStore,
	nrValues int, agentID string, thingIDs []string, timespanSec int) {

	for j := 0; j < nrValues; j++ {
		thingIndex := rand.Intn(len(thingIDs))
		thingID := thingIDs[thingIndex]
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		//randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		//randomTime := time.Now().Add(-randomSeconds)
		//ev.Created = randomTime.Format(time.RFC3339)

		// add a TD
		dThingID := td.MakeDigiTwinThingID(agentID, thingID)
		_, err := svc.ReadDThing(dThingID)
		if err != nil {
			title := fmt.Sprintf("Test thing %d", j)
			thingTD := td.NewTD(thingID, title, "randomdevice")
			dtwTD := *thingTD
			dtwTD.ID = dThingID
			svc.UpdateTD(agentID, thingTD, &dtwTD)
		}

		value := fmt.Sprintf("%2.3f", randomValue)
		svc.UpdateEventValue(agentID, thingID, valueNames[randomName], value, "")
	}
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
	var agentID = "agent1"
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, stopFunc := startLatestStore(true)

	addValues(svc, 100, agentID, thingIDs, 100)

	//valList1, err := svc.ReadLatest(vocab.HTOpPublishEvent, thingIDs[1], nil, "")
	dthingID := td.MakeDigiTwinThingID(agentID, thingIDs[1])
	valList1, err := svc.ReadAllEvents(dthingID)
	assert.NoError(t, err)
	assert.True(t, len(valList1) > 1)

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, stopFunc = startLatestStore(false)
	defer stopFunc()
	//tdList2, err := directory.ReadThings(mt, 0, 10)
	//assert.Equal(t, len(thingIDs), len(tdList2))

	dThingID1 := td.MakeDigiTwinThingID(agentID, thingIDs[1])
	valList2a, err := svc.ReadAllEvents(dThingID1)
	assert.NoError(t, err)
	assert.Equal(t, len(valList1), len(valList2a))
}

func TestGetEvents(t *testing.T) {
	const count = 100
	const agent1ID = "agent1"
	const thing1ID = "thing1"       // matches a percentage of the random things
	const valueName = "temperature" // from the list with names above
	var dThingID1 = td.MakeDigiTwinThingID(agent1ID, thing1ID)

	svc, closeFn := startLatestStore(true)
	defer closeFn()

	// add events using various names including temperature
	addValues(svc, count, agent1ID, []string{thing1ID}, 3600*24*30)

	t0 := time.Now()

	values, err := svc.ReadAllEvents(dThingID1)
	require.NoError(t, err)
	require.NotNil(t, values)
	d0 := time.Now().Sub(t0)

	// 2nd time from cache should be faster
	t1 := time.Now()
	values2, err := svc.ReadAllEvents(dThingID1)
	require.NoError(t, err)
	require.NotNil(t, values2)
	d1 := time.Now().Sub(t1)

	assert.Less(t, d1, d0, "reading from cache is not faster?")

	// reading a single event must have the same result
	value3, err := svc.ReadEvent(dThingID1, valueName)
	require.NoError(t, err)
	require.Equal(t, values2[valueName], value3)

	// save and reload props
	closeFn()
	//svc.Stop()
	svc, closeFn = startLatestStore(false)
	//err = svc.Start()
	assert.NoError(t, err)

	// LoadLatest should load and not find a cached value
	v, err := svc.ReadAllEvents(dThingID1)
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestUpdateProps(t *testing.T) {
	agent1ID := "agent1"
	thing1ID := "thing1"
	var dThingID1 = td.MakeDigiTwinThingID(agent1ID, thing1ID)
	const prop1Name = "temperature"
	const prop1Value = 10.5

	svc, closeFn := startLatestStore(true)
	defer closeFn()
	addValues(svc, 10, agent1ID, []string{thing1ID}, 3600*24*30)

	changed, err := svc.UpdatePropertyValue(agent1ID, thing1ID, prop1Name, prop1Value, "")
	require.NoError(t, err)
	require.True(t, changed)

	p1val, err := svc.ReadProperty(dThingID1, prop1Name)
	require.NoError(t, err)
	require.Equal(t, prop1Value, p1val.Data)
}

func TestAddPropsFail(t *testing.T) {
	agent1ID := "agent1"
	thing1ID := "badthingid"
	svc, closeFn := startLatestStore(true)
	_ = svc
	defer closeFn()

	changed, err := svc.UpdatePropertyValue(agent1ID, thing1ID, "prop1", "val1", "")
	require.Error(t, err)
	require.False(t, changed)
}
