package ipnet

import (
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

// var homeFolder string
var core = "mqtt"

var tempFolder string
var testServer *testenv.TestServer

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-ipnet")
	logging.SetLogging("info", "")

	//
	testServer, err = testenv.StartTestServer(core, true)
	if err != nil {
		panic("unable to start test server: " + err.Error())
	}

	result := m.Run()
	time.Sleep(time.Millisecond)
	testServer.Stop()
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
	hc, err := testServer.AddConnectUser("ipnet", authapi.ClientTypeService, authapi.ClientRoleService)
	require.NoError(t, err)
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}
