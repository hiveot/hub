package history_test

import (
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
//
//                                   ---------------  MQTT/NATS BROKER --------------
//	DBSize #Things                   kvbtree (msec)    pebble (msec)     bbolt (msec)
//	 10K       1    add 1K single        2.0            4.4          4700
//	 10K       1    add 1K batch         0.8            2.3            11
//	 10K       1    add 1K multi         1.0            2.3            11
//	 10K       1    get 1K single      310/127        313             311
//	 10K       1    get 1K batch       5.4/4.1          6               5
//
//	 10K      10    add 1K single        2.4            4.0          4600
//	 10K      10    add 1K multi         1.5            2.2            70
//	 10K      10    get 1K single      374/125        310             330
//	 10K      10    get 1K batch       7.3/4.3          6               6
//
//	100K       1    add 1K single        2.1 msec       4.2          5020
//	100K       1    add 1K batch         1.0 msec       2.3            15
//	100K       1    add 1K multi         1.0 msec       2.6            15
//	100K       1    get 1K single      328/123        380             374
//	100K       1    get 1K batch       5.6/4.4          8               6
//
//	100K      10    add 1K single        2.5            5.2          5000
//	100K      10    add 1K multi         1.1            2.3            81
//	100K      10    get 1K single      302/130        315             342
//	100K      10    get 1K batch       5.1/4.2          6               6
//
// NOTE: The get 1K test is high as it makes 1K RCP calls

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
		testData, _ := makeValueBatch("device1", tbl.dataSize, tbl.nrThings, timespanMonth)
		svc, readHist, stopFn := newHistoryService()
		time.Sleep(time.Millisecond)
		// build a dataset in the store
		addBulkHistory(svc, tbl.dataSize, 10, timespanSec)
		addHist := svc.GetAddHistory()

		// test adding records one by one
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					for i := 0; i < tbl.nrSets; i++ {
						ev := testData[i]
						err := addHist.AddEvent(ev)
						require.NoError(b, err)
					}

				}
			})
		// test adding records using the ThingID batch add for a single ThingID
		if tbl.nrThings == 1 {
			b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
				func(b *testing.B) {
					bulk := testData[0:tbl.nrSets]
					for n := 0; n < b.N; n++ {
						err := addHist.AddEvents(bulk)
						require.NoError(b, err)
					}
				})
		}
		// test adding records using the ThingID multi add for different ThingIDs
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-multi:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				bulk := testData[0:tbl.nrSets]
				for n := 0; n < b.N; n++ {
					err := addHist.AddEvents(bulk)
					require.NoError(b, err)
				}
			})

		// test reading records
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor, releaseFn, _ := readHist.GetCursor(publisherID, thing0ID, "")
					v, valid, _ := cursor.First()
					for i := 0; i < tbl.nrSets-1; i++ {
						v, valid, _ = cursor.Next()
						if !assert.True(b, valid,
							fmt.Sprintf("counting only '%d' records. Expected at least '%d'.", i, tbl.nrSets)) {
							break
						}
						assert.NotEmpty(b, v)
					}
					releaseFn()

				}
			})
		// test reading records
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor, releaseFn, _ := readHist.GetCursor(publisherID, thing0ID, "")
					require.NotNil(b, cursor)
					cursor.First()
					v, _, _ := cursor.NextN(tbl.nrSets - 1)
					if !assert.True(b, len(v) > 0,
						fmt.Sprintf("counting only '%d' records. Expected at least '%d'.", len(v), tbl.nrSets)) {
						break
					}
					assert.NotEmpty(b, v)
					releaseFn()
				}
			})

		stopFn()
		time.Sleep(time.Second) // cleanup delay
		fmt.Println("--- next round ---")
	}
}
