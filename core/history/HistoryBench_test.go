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
//                                      ---------------  MQTT/NATS BROKER --------------
//	DBSize #Things                      kvbtree (msec)    pebble (msec)     bbolt (msec)
//	 10K      10    add 1K single (*)       2.5             4.7             4600/4600
//	 10K      10    add 1K batch (*)        1.2             2.4               76/72
//	 10K      10    get 1K single         330/125         324/130            300/130
//	 10K      10    get 1K batch          5.5/4.3           7                5.5/4.3
//
//	100K      10    add 1K single (*)       2.9             4.3             4900/4900
//	100K      10    add 1K batch (*)        1.4             2.4               84/82
//	100K      10    get 1K single         340/130         322/128            325/130
//	100K      10    get 1K batch          6.0/4.2           7                5.2/4.3
//
//	  1M     100    add 1K single (*)       2.9             5.7             5500
//	  1M     100    add 1K batch (*)        1.4             3.1              580
//	  1M     100    get 1K single         310             330                303
//	  1M     100    get 1K batch          8.6               5.6                5.1
//
//	 10M       3    add 1K single (*)       3.0            10               5460
//	 10M       3    add 1K batch (*)        1.4             4.1              550
//	 10M       3    get 1K single         310             326                304
//	 10M       3    get 1K batch          6.0/4.2           6.7                5.1
//
// (*) NOTE1: adding events without message bus
//     NOTE2: get 1K single/batch test uses message bus RPC calls and are thus much higher

var DataSizeTable = []struct {
	dataSize int
	nrThings int
	nrSets   int // repeatedly set the same event
}{
	{dataSize: 10000, nrThings: 10, nrSets: 1000},
	{dataSize: 100000, nrThings: 10, nrSets: 1000},
	//{dataSize: 1000000, nrThings: 100, nrSets: 1000},
	//{dataSize: 10000000, nrThings: 100, nrSets: 1000},
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
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
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
					tv, _, _ := cursor.First()
					assert.NotEmpty(b, tv)
					if tbl.nrSets > 1 {
						tvBatch, _, _ := cursor.NextN(tbl.nrSets - 1)
						if !assert.True(b, len(tvBatch) > 0,
							fmt.Sprintf("counting only '%d' records. Expected at least '%d'.", len(tvBatch), tbl.nrSets)) {
							break
						}
					}
					releaseFn()
				}
			})

		stopFn()
		time.Sleep(time.Second) // cleanup delay
		fmt.Println("--- next round ---")
	}
}
