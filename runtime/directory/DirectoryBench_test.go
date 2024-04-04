package directory_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	"os"
	"testing"
)

// Simple performance test update/read
func Benchmark_GetTD(b *testing.B) {
	b.Log("--- Benchmark_GetTD start ---")
	defer b.Log("--- Benchmark_GetTD end ---")
	_ = os.Remove(testStoreFile)
	const senderID = "agent1"
	const thing1ID = "agent1:thing1"
	const title1 = "title1"

	logging.SetLogging("warning", "")

	// fire up the directory
	svc, stopFunc := startDirectory()
	defer stopFunc()

	// setup
	b.Run(fmt.Sprintf("update TD docs"),
		// nats: 120 usec/op
		// mqtt: 290 usec/op
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				tdDoc1 := createTDDoc(thingID, title1)
				err := svc.UpdateTD(senderID, thingID, tdDoc1)
				_ = err
			}
		})

	// test read
	b.Run(fmt.Sprintf("read TD docs"),
		// Nats: 130 usec/op
		// Mqtt: 330 usec/op
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				td, err := svc.GetTD(thingID)
				_ = td
				_ = err
			}
		})
}
