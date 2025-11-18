package ipnet

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/gocore/logging"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
)

var tempFolder string
var ts *testenv.TestServer

const agentUsesWSS = true

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	tempFolder = path.Join(os.TempDir(), "test-ipnet")
	logging.SetLogging("info", "")

	//
	ts = testenv.StartTestServer(true)

	result := m.Run()
	time.Sleep(time.Millisecond)
	ts.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}

	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const device1ID = "device1"
	cfg := config.NewIPNetConfig()
	cfg.PortScan = false
	cfg.ScanAsRoot = false

	svc := service.NewIpNetBinding(cfg)
	ag, _ := ts.AddConnectService("ipnet")
	defer ag.Disconnect()
	err := svc.Start(ag)

	require.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}
