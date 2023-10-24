package owserver_test

import (
	"encoding/json"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/hiveot/hub/plugins/owserver/config"
	"github.com/hiveot/hub/plugins/owserver/service"
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

// TestMain run mosquitto and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "./docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	logging.SetLogging("info", "")

	owsConfig = config.NewConfig()
	owsConfig.OWServerURL = owsSimulationFile

	//
	testServer, err = testenv.StartTestServer(core, true)
	if err != nil {
		panic("unable to start test server: " + err.Error())
	}

	result := m.Run()
	time.Sleep(time.Second)

	testServer.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}

	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	slog.Info("--- TestStartStop ---")
	const device1ID = "device1"

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(owsConfig, hc)
	err = svc.Start()
	assert.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const device1ID = "device1"

	slog.Info("--- TestPoll ---")
	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(owsConfig, hc)

	// Count the number of received TD events
	sub, err := hc.SubEvents("", "", "",
		func(ev *thing.ThingValue) {
			slog.Info("received event", "id", ev.Name)
			if ev.Name == vocab.EventNameProps {
				var value map[string][]byte
				err2 := json.Unmarshal(ev.Data, &value)
				assert.NoError(t, err2)
			} else {
				var value interface{}
				err2 := json.Unmarshal(ev.Data, &value)
				assert.NoError(t, err2)
			}
			tdCount.Add(1)
		})
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	// start the service which publishes TDs
	err = svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// wait until startup poll completed
	time.Sleep(time.Millisecond * 100)

	// the simulation file contains 3 things. The service is 1 thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))
}

func TestPollInvalidEDSAddress(t *testing.T) {
	slog.Info("--- TestPollInvalidEDSAddress ---")
	const device1ID = "device1"

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(owsConfig, hc)
	svc.Config.OWServerURL = "http://invalidAddress/"
	err = svc.Start()
	assert.NoError(t, err)
	defer svc.Stop()

	time.Sleep(time.Millisecond * 10)

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	slog.Info("--- TestAction ---")
	const device1ID = "device1"
	// node in test data
	const nodeID = "C100100000267C7E"
	//var nodeAddr = thing.MakeThingAddr(owsConfig.ID, nodeID)
	var actionName = vocab.VocabRelay
	var actionValue = ([]byte)("1")

	hc, err := testServer.AddConnectClient(device1ID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(owsConfig, hc)
	err = svc.Start()
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	reply, err := hc.PubAction(device1ID, nodeID, actionName, actionValue)
	assert.Error(t, err)
	_ = reply

	//time.Sleep(time.Second*10)  // debugging delay
}
