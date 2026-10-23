package owserver_test

import (
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/owserver"
	"github.com/hiveot/hub/bindings/owserver/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: switch for testing with real owserver

var tempFolder string
var storePath string
var owsConfig owserver.OWServerConfig
var owsSimulationFile string // simulation file
var testEnv *testenv.TestEnv

const agentID = "owserver"

// const device1ID = "2A000003BB170B28" // <-- from the simulation file
const device1ID = "C100100000267C7E" // <-- from the simulation file

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	var cancelFn func()
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	storePath = path.Join(tempFolder, "owserver-state")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "./docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	// uncomment the next line to discover and test with a real owserver
	//owsSimulationFile = ""
	utils.SetLogging("info", "")

	owsConfig = *owserver.NewOWServerConfig()
	owsConfig.OWServerURL = owsSimulationFile
	//
	testEnv, cancelFn = testenv.StartTestEnv("", true)
	if testEnv == nil {
		panic("unable to start test server: " + err.Error())
	}
	result := m.Run()
	time.Sleep(time.Millisecond * 10)
	println("stopping testserver")

	cancelFn()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	require.NoError(t, err)
	cc1, _ := testEnv.NewRCConnection(agentID, svc)
	defer cc1.Close()
	svc.Start()
	defer svc.Stop()

	svc.Start()
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tdCount atomic.Int32
	const userID = "user1"

	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	require.NoError(t, err)
	cc1, _ := testEnv.NewRCConnection(agentID, svc)
	defer cc1.Close()
	svc.Start()
	defer svc.Stop()

	co1, cc2, _ := testEnv.NewTestConsumer(userID, api.ClientRoleManager)
	defer co1.Stop()
	defer cc2.Close()

	// Count the number of received TD events
	err = co1.ObserveProperty("", "")
	assert.NoError(t, err)
	err = co1.Subscribe("", "")
	require.NoError(t, err)
	co1.SetNotificationHook(func(notif *msg.NotificationMessage) {
		slog.Info("received notification", "type", notif.AffordanceType, "id", notif.Name)
		var value interface{}
		err2 := notif.Decode(&value)
		assert.NoError(t, err2)

		tdCount.Add(1)
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	svc.Start()

	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 100)
	svc.Stop()

	// the simulation file contains 3 things. The service is 1 Thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))

	// get events from the digitwin
	// dThingID := digitwin.MakeDigitwinID(agentID, device1ID)
	events, err := co1.ReadAllEvents(device1ID)
	require.NoError(t, err)
	// this thing has 5 sensors and 4 alarm events
	require.Equal(t, 9, len(events))
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc, err := internal.NewOWServerBinding("", storePath, &badConfig)
	require.NoError(t, err)
	cc1, _ := testEnv.NewRCConnection(agentID, svc)
	defer cc1.Close()
	svc.Start()
	defer svc.Stop()

	svc.Start()
	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const user1ID = "operator1"
	// node in test data
	// var dThingID = digitwin.MakeDigitwinID(agentID, device1ID)
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	require.NoError(t, err)
	require.NoError(t, err)
	cc1, _ := testEnv.NewRCConnection(agentID, svc)
	defer cc1.Close()
	svc.Start()
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	co1, cc2, _ := testEnv.NewTestConnectedConsumer(user1ID, api.ClientRoleOperator, nil)
	require.NoError(t, err)
	defer cc2.Stop()
	err = co1.WriteProperty(device1ID, actionName, &actionValue, true)

	//err = co1.SendRequest(dThingID, actionName, &actionValue, nil)
	// can't write to a simulation
	assert.Error(t, err)

	//time.Sleep(time.Second * 1) // debugging delay
}

func TestConfig(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const user1ID = "manager1"
	var configName = "LEDState"
	var configValue = "1"

	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	require.NoError(t, err)
	cc1, _ := testEnv.NewRCConnection(agentID, svc)
	defer cc1.Close()
	svc.Start()
	defer svc.Stop()

	// give heartbeat time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	co1, cc2, _ := testEnv.NewTestConsumer(user1ID, api.ClientRoleManager)
	defer cc2.Close()
	// dThingID := digitwin.MakeDigitwinID(agentID, device1ID)
	err = co1.WriteProperty(device1ID, configName, &configValue, true)

	// can't write to a simulation file. Write should fail.
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
