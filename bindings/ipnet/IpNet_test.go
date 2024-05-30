package ipnet

import (
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

var tempFolder string
var ts *testenv.TestServer

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
	t.Log("--- TestStartStop ---")
	const device1ID = "device1"
	cfg := config.IPNetConfig{
		PortScan:   false,
		ScanAsRoot: false,
	}
	svc := service.NewIpNetBinding(&cfg)
	hc, _ := ts.AddConnectService("ipnet")
	defer hc.Disconnect()
	err := svc.Start(hc)

	require.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}
