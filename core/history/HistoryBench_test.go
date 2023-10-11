package history_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

const timespanHour = 3600
const timespanDay = timespanHour * 24
const timespanWeek = timespanDay * 7
const timespanMonth = timespanDay * 31
const timespanYear = timespanDay * 365

// $go test -bench=BenchmarkAddEvents -benchtime=3s -run ^#
// Performance without capnp in msec:
//
//	DBSize #Things                  kvbtree     pebble (cap)    bbolt  (cap)
//	 10K       1    add 1K single   2.5            3.6          4700
//	 10K       1    add 1K batch    2.0            2.9            11
//	 10K       1    add 1K multi    2.1            2.9            11
//	 10K       1    get 1K          1.3            2.8             1.4
//	 10K       1    get 1K batch    0.7
//
//	 10K      10    add 1K single   2.3            4.0          4800
//	 10K      10    add 1K multi    1.9            2.8            73
//	 10K      10    get 1K          1.3            1.4             1.4
//	 10K      10    get 1K batch    0.7
//
//	100K       1    add 1K single   1.9 msec       3.5          4900
//	100K       1    add 1K batch    1.6 msec       2.7            15
//	100K       1    add 1K multi    1.7 msec       2.7            14
//	100K       1    get 1K          1.2 msec       3.5             1.4
//
//	100K      10    add 1K single   2.2            3.7          4900
//	100K      10    add 1K multi    1.7            2.8            88
//	100K      10    get 1K          1.2            1.5             1.4
//
//	  1M       1    add 1K single   2.2 msec       3.5          6700
//	  1M       1    add 1K batch    1.7 msec       2.7            53
//	  1M       1    add 1K multi    1.8 msec       2.7            47
//	  1M       1    get 1K          1.2 msec       2.5             1.4
//
//	  1M     100    add 1K single   2.4            4.3          5500
//	  1M     100    add 1K multi    1.8            3.0           560
//	  1M     100    get 1K          1.2            1.4             1.4
//
// Performance with capnp:
//
//	DBSize #Things                  kvbtree     pebble (cap)    bbolt  (cap)
//	 10K       1    add 1K single     94          140            4900
//	 10K       1    add 1K batch       9.4          8.1            16
//	 10K       1    add 1K multi       9.3          8.0            16
//	 10K       1    get 1K single     91          140             140
//	 10K       1    get 1K multi       8.2          6.7           6.3
//
//	 10K      10    add 1K single     97          140            4900
//	 10K      10    add 1K multi       9.4          8.0            74
//	 10K      10    get 1K single     91          140             140
//	 10K       1    get 1K multi       8.1          6.2             6.0
//
//	100K       1    add 1K single    102          140           5200
//	100K       1    add 1K batch       9.3          7.7           20
//	100K       1    add 1K multi      10.2          7.7           20
//	100K       1    get 1K            91          140            140
//	100K       1    get 1K multi       7.8          8.2            5.7
//
//	100K      10    add 1K single    100          140           5200
//	100K      10    add 1K multi       9.6          7.8           88
//	100K      10    get 1K            92          140            140
//	100K      10    get 1K multi       8.6          5.9            5.6
//
//	  1M       1    add 1K single     90          140           6900
//	  1M       1    add 1K batch      12            7.8           58
//	  1M       1    add 1K multi       8.8          7.6           48
//	  1M       1    get 1K            92          140            140
//	  1M       1    get 1K multi       7.8          6.5            5.6
//
//	  1M     100    add 1K single     99          140           5900
//	  1M     100    add 1K multi      12            7.6          540
//	  1M     100    get 1K            78          136            140
//	  1M     100    get 1K multi       8.2          5.7            5.6
var DataSizeTable = []struct {
	dataSize int
	nrThings int
	nrSets   int
}{
	{dataSize: 10000, nrThings: 1, nrSets: 1000},
	{dataSize: 10000, nrThings: 10, nrSets: 1000},
	{dataSize: 100000, nrThings: 1, nrSets: 1000},
	{dataSize: 100000, nrThings: 10, nrSets: 1000},
	// {dataSize: 1000000, nrThings: 1, nrSets: 1000},
	// {dataSize: 1000000, nrThings: 100, nrSets: 1000},
}

func BenchmarkAddEvents(b *testing.B) {
	const publisherID = "device1"
	const thing0ID = thingIDPrefix + "0"
	const timespanSec = 3600 * 24 * 10

	logging.SetLogging("error", "")

	for _, tbl := range DataSizeTable {
		ctx := context.Background()
		testData, _ := makeValueBatch("device1", tbl.dataSize, tbl.nrThings, timespanMonth)
		svc, closeFn := newHistoryService(useTestCapnp)
		// build a dataset in the store
		addHistory(svc, tbl.dataSize, 10, timespanSec)

		updateHistory, _ := svc.CapAddHistory(ctx, "test", true)
		readHistory, _ := svc.CapReadHistory(ctx, "test")

		// test adding records one by one
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					for i := 0; i < tbl.nrSets; i++ {
						ev := testData[i]
						err := updateHistory.AddEvent(ctx, ev)
						require.NoError(b, err)
					}

				}
			})
		// test adding records using the ThingID batch add
		if tbl.nrThings == 1 {
			b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
				func(b *testing.B) {
					bulk := testData[0:tbl.nrSets]
					for n := 0; n < b.N; n++ {
						err := updateHistory.AddEvents(ctx, bulk)
						require.NoError(b, err)
					}
				})
		}
		// test adding records using the ThingID multi add of different IDs
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-multi:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				bulk := testData[0:tbl.nrSets]
				for n := 0; n < b.N; n++ {
					err := updateHistory.AddEvents(ctx, bulk)
					require.NoError(b, err)
				}
			})

		// test reading records
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor := readHistory.GetEventHistory(ctx, publisherID, thing0ID, "")
					require.NotNil(b, cursor)
					cursor.First()
					for i := 0; i < tbl.nrSets-1; i++ {
						v, valid := cursor.Next()
						if !assert.True(b, valid,
							fmt.Sprintf("counting only '%d' records. Expected at least '%d'.", i, tbl.nrSets)) {
							break
						}
						assert.NotEmpty(b, v)
					}
					cursor.Release()

				}
			})
		// test reading records
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor := readHistory.GetEventHistory(ctx, publisherID, thing0ID, "")
					require.NotNil(b, cursor)
					cursor.First()
					v, _ := cursor.NextN(uint(tbl.nrSets - 1))
					if !assert.True(b, len(v) > 0,
						fmt.Sprintf("counting only '%d' records. Expected at least '%d'.", len(v), tbl.nrSets)) {
						break
					}
					assert.NotEmpty(b, v)
					cursor.Release()
				}
			})
		updateHistory.Release()
		readHistory.Release()

		//b.Log("- next round -")
		time.Sleep(time.Second) // cleanup delay
		fmt.Println("--- next round ---")
		closeFn()

	}
}
