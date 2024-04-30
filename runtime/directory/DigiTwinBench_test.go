package directory_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/lib/logging"
	"testing"
)

// Simple performance test update/read
// Benchmark_GetTD/update_TD_docs-4    4768 ns/op
// Benchmark_GetTD/read_TD_docs-4       739.4 ns/op
func Benchmark_ReadTD(b *testing.B) {
	b.Log("--- Benchmark_ReadTD start ---")
	defer b.Log("--- Benchmark_GetTD end ---")
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	logging.SetLogging("warning", "")

	// fire up the directory
	svc, cl, stopFunc := startDirectory(true)
	_ = cl
	defer stopFunc()

	// setup
	b.Run(fmt.Sprintf("update TD docs"),
		// old values kept for future comparison:
		// nats: 120 usec/op
		// mqtt: 290 usec/op
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				tdDoc1 := createTDDoc(thingID, title1)
				err := svc.UpdateThing(senderID, thingID, tdDoc1)
				_ = err
			}
		})

	// test read
	b.Run(fmt.Sprintf("read TD docs"),
		// old values kept for future comparison:
		// Nats: 130 usec/op
		// Mqtt: 330 usec/op
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				td, err := svc.ReadThing(directory.ReadThingArgs{ThingID: thingID})
				_ = td
				_ = err
			}
		})
}
