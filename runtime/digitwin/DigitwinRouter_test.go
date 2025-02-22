package digitwin_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/runtime/digitwin/router"
	"os"
	"path"
	"testing"
)

var testDir = path.Join(os.TempDir(), "test-router")

// runtime tests the router

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

// just a compile check
func TestStartStop(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	// API match check
	var _ router.IDigitwinRouter = &router.DigitwinRouter{}
	//r := hubrouter.NewDigitwinRouter(dtwService, dirAgent, authnAction, authzAction)

}
