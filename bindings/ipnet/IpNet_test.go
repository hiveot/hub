package ipnet

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
	"github.com/stretchr/testify/require"
)

var tempFolder string
var testEnv *testenv.TestEnv

const agentUsesWSS = true

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	var stopFn func()
	// setup environment
	tempFolder = path.Join(os.TempDir(), "test-ipnet")
	utils.SetLogging("info", "")

	//
	testEnv, stopFn = testenv.StartTestEnv("")

	result := m.Run()
	time.Sleep(time.Millisecond)
	stopFn()
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
	cc1, _ := testEnv.NewConnectedClient("ipnet", authn.ClientRoleService)
	defer cc1.Close()
	err := svc.Start(cc1)

	require.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}
