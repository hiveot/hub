package hubrouter_test

import (
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/hubrouter"
	"os"
	"path"
	"testing"
)

var testDir = path.Join(os.TempDir(), "test-router")

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	_ = os.RemoveAll(testDir)
	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	// API match check
	var _ hubrouter.IHubRouter = &hubrouter.HubRouter{}
}
