package ipnet

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/bindings/ipnet/service"
)

var tempFolder string
var testEnv *testenv.TestEnv

const agentUsesWSS = true

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {

	// setup environment
	tempFolder = path.Join(os.TempDir(), "test-ipnet")
	utils.SetLogging("info", "")
	testEnv = testenv.NewTestEnv(true)

	result := m.Run()
	time.Sleep(time.Millisecond)

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
	defer svc.Stop()
	// create and link the consumer
	co1 := consumer.NewConsumer(svc, nil)
	defer co1.Stop()
	svc.Start()
	co1.Start()

	time.Sleep(time.Second)
}
