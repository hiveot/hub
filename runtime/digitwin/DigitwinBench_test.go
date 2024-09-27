package digitwin_test

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkUpdateDTW(b *testing.B) {
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	const title1 = "title1"

	logging.SetLogging("warning", "")
	svc, store, stopFn := startService(true)
	store2 := store
	defer stopFn()

	var err error

	// json: 8800 us
	// json-iterator: 10800 us
	// shamaton/msgpack: 12000 us
	// vmihailenco/msgpack/v5: 10700 us
	b.Run(fmt.Sprintf("update DTW"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				for m := range 1000 {
					thingID := fmt.Sprintf("%s-%d", thing1ID, m)
					tdDoc := createTDDoc(thingID, 5, 4, 3)
					tdDocJson, _ := json.Marshal(tdDoc)
					err = svc.UpdateTD(agent1ID, thingID, string(tdDocJson))
					assert.NoError(b, err)
				}
			}
		})

	// json: 13600 us
	// json-iterator: 7030 us
	// shamaton/msgpack: 9400 us
	// vmihailenco/msgpack/v5: 11600
	b.Run(fmt.Sprintf("LoadCacheFromStore"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				err = store2.LoadCacheFromStore()
				assert.NoError(b, err)
			}
		})
	assert.NoError(b, err)
}
