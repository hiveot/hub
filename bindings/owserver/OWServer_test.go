package owserver_test

import (
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

// var homeFolder string
var core = "mqtt"

var tempFolder string
var owsConfig config.OWServerConfig
var owsSimulationFile string // simulation file
var testServer *testenv.TestServer

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "./docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	logging.SetLogging("info", "")

	owsConfig = *config.NewConfig()
	owsConfig.OWServerURL = owsSimulationFile
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

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)
	err = svc.Start(hc)
	assert.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const device1ID = "device1"

	t.Log("--- TestPoll ---")
	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)

	// Count the number of received TD events
	err = hc.SubEvents("", "", "")
	require.NoError(t, err)
	hc.SetEventHandler(func(ev *things.ThingValue) {
		slog.Info("received event", "id", ev.Name)
		if ev.Name == vocab.EventNameProps {
			var value map[string]interface{}
			err2 := ser.Unmarshal(ev.Data, &value)
			assert.NoError(t, err2)
		} else {
			var value interface{}
			err2 := ser.Unmarshal(ev.Data, &value)
			assert.NoError(t, err2)
		}
		tdCount.Add(1)
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// wait until startup poll completed
	time.Sleep(time.Millisecond * 100)

	// the simulation file contains 3 things. The service is 1 things.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Log("--- TestPollInvalidEDSAddress ---")
	const device1ID = "device1"

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc := service.NewOWServerBinding(&badConfig)
	err = svc.Start(hc)
	assert.NoError(t, err)
	defer svc.Stop()

	time.Sleep(time.Millisecond * 10)

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Log("--- TestAction ---")
	const device1ID = "device1"
	const user1ID = "operator1"
	// node in test data
	const nodeID = "C100100000267C7E"
	//var nodeAddr = things.MakeThingAddr(owsConfig.ID, nodeID)
	var actionName = vocab.VocabRelay
	var actionValue = ([]byte)("1")

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, err := testServer.AddConnectClient(user1ID, authapi.ClientTypeUser, authapi.ClientRoleOperator)
	require.NoError(t, err)
	defer hc2.Disconnect()
	reply, err := hc.PubAction(device1ID, nodeID, actionName, actionValue)
	assert.Error(t, err)
	_ = reply

	//time.Sleep(time.Second*10)  // debugging delay
}

func TestConfig(t *testing.T) {
	t.Log("--- TestConfig ---")
	const device1ID = "device1"
	const user1ID = "manager1"
	// node in test data
	const nodeID = "C100100000267C7E"
	//var nodeAddr = things.MakeThingAddr(owsConfig.ID, nodeID)
	var configName = "LEDFunction"
	var configValue = ([]byte)("1")

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, err := testServer.AddConnectClient(user1ID, authapi.ClientTypeUser, authapi.ClientRoleManager)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.PubConfig(device1ID, nodeID, configName, configValue)
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
