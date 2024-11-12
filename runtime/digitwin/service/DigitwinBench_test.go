package service_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkUpdateDTW(b *testing.B) {
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const count = 1000

	// create the test TDs
	tddocs := make([]string, 0)
	for m := range 1000 {
		thingID := fmt.Sprintf("%s-%d", thing1ID, m)
		tdDoc := createTDDoc(thingID, 20, 10, 3)
		tdDocJson, _ := jsoniter.Marshal(tdDoc)
		//tdDocJson, _ := msgpack.Marshal(tdDoc)
		tddocs = append(tddocs, string(tdDocJson))
	}
	logging.SetLogging("warning", "")
	svc, store, stopFn := startService(true)
	store2 := store
	dirSvc := svc.DirSvc
	defer stopFn()

	var err error

	// 160 msec using golang json to update 1000 things
	// 78 msec using json-iterator (jsoniter)
	b.Run(fmt.Sprintf("update DTW"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				for m := range 1000 {
					tdDocJson := tddocs[m]
					err = dirSvc.UpdateTD(agent1ID, string(tdDocJson))
					assert.NoError(b, err)
				}
			}
		})

	// 165 msec to reload a cache with 1000 things using std json
	// 78 msec with json-iterator
	b.Run(fmt.Sprintf("LoadCacheFromStore"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				err = store2.LoadCacheFromStore()
				assert.NoError(b, err)
			}
		})
	assert.NoError(b, err)
}
