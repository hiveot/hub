package buckets_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thanhpk/randstr"

	"github.com/hiveot/hub/lib/logging"
)

// $ go test -bench=Benchmark_bucket -benchtime=3s -run=^#    (skip unit tests)
// cpu: Intel(R) Core(TM) i5-4570S CPU @ 2.90GHz
//
//	Database:                kvbtree (us)      pebble (us)               bbolt (us)
//	--- with 1K records existing in the DB ---
//	Set 1                       0.3              1.9                       4900
//	Set Multiple 1              0.5              2.2                       5000
//	Get 1                       0.3              0.7                          1.1
//	Get Multiple 1              0.9              1.9                          1.5
//	Seek 1                      0.9              3.2                          1.6
//	Next 1                      1.1            452 (163-450)                  1.7
//	Set 1000x                 170             2250 (2100-2300)          5200000  (5 sec!)
//	Set Multiple 1000         280             2100                        10000
//	Get 1000x                 140              710 ( 710-1000)             1180
//	Get Multiple 1000         260             1200 (1200-1400)              740
//	Seek 1000x                140             1540 (1540-2500)              280
//	Next 1000x                140              420 ( 420-1310)              190
//
//	--- with 100K existing records in DB ---
//	Set 1                       0.4              2.0                      11000
//	Set Multiple 1              0.6              2.2                      11700
//	Get 1                       0.3              0.7                          1.2
//	Get Multiple 1              1.0              1.9                          1.4
//	Seek 1                      1.0              2.7                          1.7
//	Next 1                      1.1           1190 (3.5-1300?)                1.7
//	Set 1000x                 230             2700 (2700-2900)         12000000   (12sec!???)
//	Set Multiple 1000         360             2600                        14300
//	Get 1000x                 180              760 ( 690-1600)             1238
//	Get Multiple 1000         300             1310                          800
//	Seek 1000x                180             2600 (1800-2600)              340
//	Next 1000x                140              640 ( 370-1400)              210
//
// Observations:
//   - kvbtree, an in-memory btree, is the overall winner, although this is far from a complete picture
//   - pebble has great real-life performance and scales much further than kvbtree
//     the next1 oddball might be due to locking delays because of the testcase as next 1000x is faster.
//   - bbolt does well on reading and iterating but is slow on write, especially with large datasets.
//     Maybe this is caused by a problem in the bucket store implementation though.
//   - an RPC call is around 100 usec on this machine, so except for bbolt this is much slower than single store operations
//   - setmultiple is slightly slower (due to building test keys) but it will much(!) faster than making N rpc calls
//
// table with data size to run the benchmark with
var DataSizeTable = []struct {
	dataSize int
	nrSteps  int
}{
	{dataSize: 1000, nrSteps: 1},
	{dataSize: 1000, nrSteps: 1000},
	{dataSize: 100000, nrSteps: 1},
	{dataSize: 100000, nrSteps: 1000},
	//{dataSize: 1000000, textSize: 100},
	//{dataSize: 10000000, textSize: 100},
}

// Generate random test data used to set and set multiple
type TestEl struct {
	key string
	val []byte
}

var testData = func() []TestEl {
	count := 1000000
	keySize := 10   // 10 char string
	textSize := 100 // 100 byte data
	data := make([]TestEl, count)
	for i := 0; i < count; i++ {
		key := randstr.String(keySize)
		val := randstr.Bytes(textSize)
		data[i] = TestEl{key: key, val: val}
	}
	return data
}()

func Benchmark_bucket(b *testing.B) {
	logging.SetLogging("warning", "")

	for _, v := range DataSizeTable {
		//setup
		//testText := randstr.String(v.textSize)
		store, _ := openNewStore()
		err := addDocs(store, testBucketID, v.dataSize)
		assert.NoError(b, err)

		// bucket.Set
		b.Run(fmt.Sprintf("Bucket.Set datasize=%d;steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					bucket := store.GetBucket(testBucketID)

					for i := 0; i < v.nrSteps; i++ {
						td := testData[i]
						err = bucket.Set(td.key, td.val)
						assert.NoError(b, err)
					}
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})
		// bucket.SetMultiple
		b.Run(fmt.Sprintf("Bucket.SetMultiple datasize=%d;steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					bucket := store.GetBucket(testBucketID)
					docs := make(map[string][]byte)
					for i := 0; i < v.nrSteps; i++ {
						td := testData[i]
						docs[td.key] = td.val
					}
					err = bucket.SetMultiple(docs)
					assert.NoError(b, err)
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})
		// bucket.Get
		b.Run(fmt.Sprintf("Bucket.Get datasize=%d,;steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					bucket := store.GetBucket(testBucketID)
					for i := 0; i < v.nrSteps; i++ {
						td := testData[i]
						_, err := bucket.Get(td.key)
						assert.NoError(b, err)
					}
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})
		// bucket.GetMultiple
		b.Run(fmt.Sprintf("Bucket.GetMultiple datasize=%d,;steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					bucket := store.GetBucket(testBucketID)
					keys := make([]string, v.nrSteps)
					for i := 0; i < v.nrSteps; i++ {
						td := testData[i]
						keys[i] = td.key
					}
					docs, err := bucket.GetMultiple(keys)
					assert.Equal(b, len(keys), len(docs))
					assert.NoError(b, err)
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})
		// bucket.Seek
		b.Run(fmt.Sprintf("Bucket.Seek datasize=%d,steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					bucket := store.GetBucket(testBucketID)
					cursor := bucket.Cursor(nil)

					// cursor based seek (find nearest) instead of a get
					for i := 0; i < v.nrSteps; i++ {
						td := testData[i]
						key2, val2, valid := cursor.Seek(td.key)
						_ = key2
						_ = val2
						assert.True(b, valid)
						assert.NoError(b, err)
					}

					cursor.Release()
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})
		// bucket.Next (range)
		b.Run(fmt.Sprintf("Bucket.Next datasize=%d,steps=%d", v.dataSize, v.nrSteps),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					bucket := store.GetBucket(testBucketID)
					cursor := bucket.Cursor(nil)
					k0, v0, valid0 := cursor.First()
					assert.True(b, valid0)
					assert.NotEmpty(b, k0)
					assert.NotEmpty(b, v0)

					// cursor based iteration
					for i := 0; i < v.nrSteps; i++ {
						k1, v1, valid1 := cursor.Next()
						assert.True(b, valid1)
						assert.NotEmpty(b, k1)
						assert.NotEmpty(b, v1)
					}

					cursor.Release()
					err = bucket.Close()
					assert.NoError(b, err)
				}
			})

		err = store.Close()
		assert.NoError(b, err)
	}
}
