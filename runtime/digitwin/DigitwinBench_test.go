package digitwin_test

import (
	"fmt"
	"testing"

	"github.com/hiveot/hivekit/go/logging"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func BenchmarkUpdateDTW(b *testing.B) {
	b.Logf("---%s---\n", b.Name())
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"
	const count = 1000

	// create the test TDs
	tddocs := make([]string, 0)
	for m := range 1000 {
		thingID := fmt.Sprintf("%s-%d", thing1ID, m)
		tdDoc := createTDDoc(thingID, 20, 10, 3)
		tdDocJson, _ := jsoniter.MarshalToString(tdDoc)
		tddocs = append(tddocs, tdDocJson)
	}
	logging.SetLogging("warning", "")
	svc, store, stopFn := startService(true)
	store2 := store
	dirSvc := svc.DirSvc
	defer stopFn()

	var err error

	// 140 msec to update 1000 things
	b.Run("update DTW",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				for m := range 1000 {
					tdDocJson := tddocs[m]
					err = dirSvc.UpdateThing(agent1ID, string(tdDocJson))
					assert.NoError(b, err)
				}
			}
		})

	// 78 msec to reload a cache with 1000 things (using json-iterator)
	b.Run("LoadCacheFromStore",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				err = store2.LoadCacheFromStore()
				assert.NoError(b, err)
			}
		})
	assert.NoError(b, err)
}
