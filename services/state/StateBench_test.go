package state_test

import (
	"fmt"
	"github.com/hiveot/hub/services/state/stateclient"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"testing"

	"github.com/thanhpk/randstr"

	"github.com/hiveot/hub/lib/logging"
)

// Add records to the state store
func addRecords(stateCl *stateclient.StateClient, count int) {
	const batchSize = 1405
	nrBatches := (count / batchSize) + 1

	// Don't exceed the max transaction size
	for iBatch := 0; iBatch < nrBatches; iBatch++ {

		docs := make(map[string]string)
		for i := 0; i < batchSize && count > 0; i++ {
			k := randstr.String(12)
			v := randstr.String(100)
			docs[k] = v
			count--
		}
		err := stateCl.SetMultiple(docs)
		if err != nil {
			log.Panicf("set multiple failed with: %s", err)
		}
		//client.Commit(nil)
	}
}

// Benchmark with use of capnp. Note timing in msec
//
//              DB size   records        kv (ms)       pebble (ms)     bbolt (ms)
// SetState        1K,       1             0.1            0.1              5.0
//               100K,       1             0.1            0.1              7.1
//                 1K      1000          120            140             4900   (4.9 sec!)
//               100K      1000          120            140             7000   (7 sec!)
// SetMultiple     1K,       1             0.14           0.15             5.1
//               100K,       1             0.13           0.15             6.9
//                 1K      1000            4.3            6.6             16
//               100K      1000            4.3            7.9             36
// GetState        1K,       1             0.13           0.13             0.13
//               100K,       1             0.13           0.13             0.13
//                 1K      1000          130            130              130
//               100K      1000          130            130              130

// Observations:
//   - transaction write of bbolt is very costly. Use setmultiple or performance will be insufficient
//   - the capnp RPC over Unix Domain Sockets call overhead is around 0.13 msec.
//
// With the digitwin http RPC transport (round trip)
//   - the http RPC over http call overhead is around 0.8 msec, so numbers are 8 times higher
var DataSizeTable = []struct {
	dataSize int
	nrSets   int
}{
	{dataSize: 1000, nrSets: 1},
	//{dataSize: 100000, nrSets: 1},
	{dataSize: 1000, nrSets: 1000},
	//{dataSize: 100000, nrSets: 1000},
	//{dataSize: 1000000, nrSets: 1},
	//{dataSize: 1000000, nrSets: 1000},
}

// Generate random test data used to set and set multiple
type TestEl struct {
	key string
	val string
}

func makeTestData() []TestEl {
	count := 100000
	data := make([]TestEl, count)
	for i := 0; i < count; i++ {
		key := randstr.String(10)  // 10 char string
		val := randstr.String(100) // 100 byte data
		data[i] = TestEl{key: key, val: val}
	}
	return data
}

// test performance of N random set state
func BenchmarkSetState(b *testing.B) {

	// setup
	logging.SetLogging("warning", "")
	testData := makeTestData()

	for _, tbl := range DataSizeTable {

		svc, stateCl, stopFn := startStateService(true)
		_ = svc
		logging.SetLogging("warning", "")

		addRecords(stateCl, tbl.dataSize)

		b.Run(fmt.Sprintf("SetState. Datasize=%d, #sets=%d", tbl.dataSize, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					// range iterator adds approx 0.4 usec per call for 1M dataset
					for i := 0; i < tbl.nrSets; i++ {
						j := rand.Intn(100000)
						td := testData[j]
						err := stateCl.Set(td.key, td.val)
						//err := svc.Set("user1", td.key, td.val)
						assert.NoError(b, err)
					}
				}
			})
		stopFn()
	}
}

// test performance of N random set state
func BenchmarkSetMultiple(b *testing.B) {
	testData := makeTestData()
	logging.SetLogging("warning", "")

	for _, tbl := range DataSizeTable {
		// setup
		_, stateCl, stopFn := startStateService(true)
		logging.SetLogging("warning", "")
		addRecords(stateCl, tbl.dataSize)

		// build a set of data to test with
		multiple := make(map[string]string)
		_ = multiple
		for i := 0; i < tbl.nrSets; i++ {
			td := testData[i]
			multiple[td.key] = string(td.val)
		}

		b.Run(fmt.Sprintf("SetMultiple. Datasize=%d, #sets=%d", tbl.dataSize, tbl.nrSets),
			func(b *testing.B) {
				// test set
				for n := 0; n < b.N; n++ {
					err := stateCl.SetMultiple(multiple)
					assert.NoError(b, err)
				}
			})
		stopFn()
	}
}

// test performance of N single get/sets
func BenchmarkGetState(b *testing.B) {
	const key1 = "key1"
	var val1 = "value 1"
	logging.SetLogging("warning", "")

	for _, tbl := range DataSizeTable {
		// setup
		_, stateCl, stopFn := startStateService(true)
		logging.SetLogging("warning", "")
		addRecords(stateCl, tbl.dataSize)

		err := stateCl.Set(key1, val1)
		assert.NoError(b, err)

		// create the client, update and close
		b.Run(fmt.Sprintf("GetState. Datasize=%d, %d gets",
			tbl.dataSize, tbl.nrSets), func(b *testing.B) {

			// test get
			for n := 0; n < b.N; n++ {
				assert.NoError(b, err)
				for i := 0; i < tbl.nrSets; i++ {
					val2 := ""
					found, err2 := stateCl.Get(key1, &val2)
					assert.True(b, found)
					assert.NoError(b, err2)
					assert.Equal(b, val1, val2)
				}
			}
		})
		stopFn()
	}
}
