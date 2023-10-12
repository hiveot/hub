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
	const publisherID = "urn:test"
	const thing1ID = "urn:thing1"
	const title1 = "title1"

	logging.SetLogging("warning", "")

	// fire up the directory
	rd, up, stopFunc := startDirectory()
	_ = up
	defer stopFunc()

	// setup
	b.Run(fmt.Sprintf("setup. creating TD docs"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				tdDoc1 := createTDDoc(thingID, title1)
				err := up.UpdateTD(publisherID, thingID, tdDoc1)
				_ = err
			}
		})

	// test read
	b.Run(fmt.Sprintf("reading TD docs"),
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				thingID := fmt.Sprintf("%s-%d", thing1ID, n)
				td, err := rd.GetTD(publisherID, thingID)
				_ = td
				_ = err
			}
		})
}
