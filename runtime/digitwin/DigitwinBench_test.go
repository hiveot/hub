package digitwin_test

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

	// 80000 usec to update a thing using std json unmarshaller
	// 43000 usec using json-iterator (jsoniter)
	// 95000 usec using shamaton/msgpack
	b.Run(fmt.Sprintf("update DTW"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				for m := range 1000 {
					tdDocJson := tddocs[m]
					err = dirSvc.UpdateDTD(agent1ID, string(tdDocJson))
					assert.NoError(b, err)
				}
			}
		})

	// 158000 usec to reload a cache with 1000 things using std json
	//  76000 usec with json-iterator
	// 189000 usec with shamaton/msgpack
	b.Run(fmt.Sprintf("LoadCacheFromStore"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				err = store2.LoadCacheFromStore()
				assert.NoError(b, err)
			}
		})
	assert.NoError(b, err)
}
