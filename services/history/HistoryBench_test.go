package history_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hiveot/gocore/wot/td"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/gocore/logging"
)

const timespanHour = 3600
const timespanDay = timespanHour * 24
const timespanWeek = timespanDay * 7
const timespanMonth = timespanDay * 31
const timespanYear = timespanDay * 365

// $go test -bench=BenchmarkAddEvents -benchtime=3s -run ^#
//                                                                                          --- WOT protocol bindings (reading) ---
//                                      ---------------  MQTT/NATS BROKER --------------     HTTPSSE runtime          WSS runtime
//	DBSize #Things                      kvbtree (msec)    pebble (msec)     bbolt (msec)      pebble (msec)           pebble (msec)
//	 10K      10    add 1K single (*)       2.5             4.7             4600/4600             5.6                    6.1           <- increase of 40% due to internal changes
//	 10K      10    add 1K batch (*)        1.2             2.4               76/72               5.6                    3.5
//	 10K      10    get 1K single         330/125         324/130            300/130            770     (!! ouch)      190
//	 10K      10    get 1K batch          5.5/4.3           7                5.5/4.3             17     (!!)            15
//
//	100K      10    add 1K single (*)       2.9             4.3             4900/4900             6.7                    5.5
//	100K      10    add 1K batch (*)        1.4             2.4               84/82               5.8                    3.1
//	100K      10    get 1K single         340/130         320/128            325/130            750     (!! ouch)      190
//	100K      10    get 1K batch          6.0/4.2           7                5.2/4.3             15                     16
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
//     NOTE3: the new digitwin runtime using HTTP/SSE is half as fast, as expected, but the get 1K single is horrendous!
//            looks like crypto is taking the bulk of it. Is it the re-auth?

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
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0"
	const timespanSec = 3600 * 24 * 10
	var dThing0ID = td.MakeDigiTwinThingID(agentID, thing0ID)

	logging.SetLogging("error", "")

	for _, tbl := range DataSizeTable {
		testData, _ := makeValueBatch("device1", tbl.dataSize, tbl.nrThings, timespanMonth)
		svc, readHist, stopFn := startHistoryService(true)
		time.Sleep(time.Millisecond)
		// build a dataset in the store
		addBulkHistory(svc, agentID, tbl.dataSize, 10, timespanSec)

		addHist := svc.GetAddHistory()

		// test adding records one by one
		// add history directly access the history store. No comms protocol is used.
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					for i := 0; i < tbl.nrSets; i++ {
						ev := testData[i]
						err := addHist.AddValue(agentID, ev)
						require.NoError(b, err)
					}

				}
			})
		// test adding records using the ThingID batch add for a single ThingID
		// add history directly access the history store. No comms protocol is used.
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d add-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				bulk := testData[0:tbl.nrSets]
				for n := 0; n < b.N; n++ {
					for _, v := range bulk {
						err := addHist.AddValue(agentID, v)
						require.NoError(b, err)
					}
				}
			})

		// test reading records
		// readHist uses the hubclient library
		time.Sleep(time.Millisecond * 300) // let the add settle
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-single:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor, releaseFn, _ := readHist.GetCursor(dThing0ID, "")
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
		// readHist uses the hubclient library
		b.Run(fmt.Sprintf("[dbsize:%d] #things:%d get-batch:%d", tbl.dataSize, tbl.nrThings, tbl.nrSets),
			func(b *testing.B) {
				for n := 0; n < b.N; n++ {

					cursor, releaseFn, _ := readHist.GetCursor(dThing0ID, "")
					require.NotNil(b, cursor)
					tv, _, _ := cursor.First()
					assert.NotEmpty(b, tv)
					if tbl.nrSets > 1 {
						tvBatch, _, _ := cursor.NextN(tbl.nrSets-1, "")
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
