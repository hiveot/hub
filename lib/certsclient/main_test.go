package certsclient_test

import (
	"os"
	"path"
	"testing"

	"github.com/hiveot/hub/lib/logging"
)

var TestCertFolder string
var testPrivKeyPemFile string
var testPubKeyPemFile string

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	TestCertFolder, _ = os.MkdirTemp("", "hiveot-go-")

	testPrivKeyPemFile = path.Join(TestCertFolder, "privKey.pem")
	testPubKeyPemFile = path.Join(TestCertFolder, "pubKey.pem")
	logging.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", TestCertFolder)
	} else {
		// comment out the next line to be able to inspect results
		_ = os.RemoveAll(TestCertFolder)
	}

	os.Exit(result)
}
