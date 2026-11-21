package owserver_test

import (
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/owserver"
	"github.com/hiveot/hub/bindings/owserver/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: switch for testing with real owserver

var tempFolder string
var owsConfig owserver.OWServerConfig
var owsSimulationFile string // simulation file
var testEnv *testenv.TestEnv
var storePath string

// --- set liveTest to discover and test with a real owserver ---
const liveTest = false

// const agentID = "owserver"

// const device1ID = "2A000003BB170B28" // <-- from the simulation file
const device1ID = "C100100000267C7E" // <-- from the simulation file

// TestMain creates the test environment.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	cwd, _ := os.Getwd()
	testFolder := path.Join(cwd, "./")
	owsSimulationFile = "file://" + path.Join(testFolder, "owserver-simulation.xml")

	if liveTest {
		owsSimulationFile = ""
	}
	utils.SetLogging("info", "")

	owsConfig = *owserver.NewOWServerConfig()
	owsConfig.OWServerURL = owsSimulationFile
	testEnv = testenv.NewTestEnv(true)
	storePath = testEnv.Env.GetStorageDir(owserver.OWServerCellType)

	result := m.Run()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := internal.NewOWServerGateway("", storePath, &owsConfig)
	defer svc.Stop()
	require.NoError(t, err)
	// link the test consumer directly to the owserver cell.
	co1 := consumer.NewConsumer("owserver-test", nil, svc)
	defer co1.Stop()

	// cc1, _ := testEnv.NewRCConnection(agentID, svc)
	// defer cc1.Close()
	svc.Start()
	co1.Start()

	svc.Start()
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var notifCount atomic.Int32
	const userID = "user1"

	svc, err := internal.NewOWServerGateway("", storePath, &owsConfig)

	// co1 will receive all notifications
	co1 := consumer.NewConsumer("owserver-test", func(notif *msg.NotificationMessage) {
		notifCount.Add(1)
	}, svc)

	svc.Start()
	defer svc.Stop()
	defer co1.Stop()

	// give heartbeat a chance to run. stop will wait for it to complete

	// get events that were published
	events, err := co1.ReadAllEvents(device1ID)
	require.NoError(t, err)

	// time.Sleep(time.Millisecond * 300)
	assert.Greater(t, int(notifCount.Load()), 10)

	// this thing has 5 sensors and 4 alarm events
	require.Equal(t, 9, len(events))

}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc, err := internal.NewOWServerGateway("", storePath, &badConfig)
	svc.Start()
	defer svc.Stop()

	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	_, err = svc.UpdateThings(false)
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const user1ID = "operator1"
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	// increase timeout for debugging
	msg.CellTimeout = time.Minute * 2

	svc, err := internal.NewOWServerGateway("", storePath, &owsConfig)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	// note that the simulation file doesn't support writes so this logs an error
	slog.Error("--- Expect an error below. Can't write to simulation file ---")
	co1 := consumer.NewConsumer("testaction", nil, svc)

	err = co1.WriteProperty(device1ID, actionName, &actionValue, true)
	if liveTest {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}
}

func TestConfig(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const user1ID = "manager1"
	var configName = "LEDState"
	var configValue = "1"

	svc, err := internal.NewOWServerGateway("", storePath, &owsConfig)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	// give heartbeat time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	slog.Error("--- Expect an error below. Can't write to simulation file ---")
	co1 := consumer.NewConsumer("testConfig", nil, svc)
	err = co1.WriteProperty(device1ID, configName, &configValue, true)
	if liveTest {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}
}
