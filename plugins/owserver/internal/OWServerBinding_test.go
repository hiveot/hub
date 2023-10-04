package internal_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/testenv"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/plugins/owserver/internal"
)

// var homeFolder string
var core = "mqtt"

var tempFolder string
var owsConfig internal.OWServerConfig
var owsSimulationFile string // simulation file
var testServer *testenv.TestServer

// TestMain run mosquitto and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "../docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	logging.SetLogging("info", "")

	owsConfig = internal.NewConfig()
	owsConfig.OWServerURL = owsSimulationFile

	//
	testServer, err = testenv.StartTestServer(core)
	if err != nil {
		panic("unable to start test server: " + err.Error())
	}
	testServer.StartAuth()

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

	hc, err := testServer.AddConnectClient(device1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := internal.NewOWServerBinding(owsConfig, hc)
	err = svc.Start()
	assert.NoError(t, err)
	defer svc.Stop()
	time.Sleep(time.Second)
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const device1ID = "device1"

	slog.Info("--- TestPoll ---")
	hc, err := testServer.AddConnectClient(device1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()
	svc := internal.NewOWServerBinding(owsConfig, hc)

	// Count the number of received TD events
	sub, err := hc.SubEvents("", "",
		func(ev *hubclient.EventMessage) {
			slog.Info("received event", "id", ev.EventID)
			if ev.EventID == vocab.EventNameProps {
				var value map[string][]byte
				err2 := json.Unmarshal(ev.Payload, &value)
				assert.NoError(t, err2)
			} else {
				var value interface{}
				err2 := json.Unmarshal(ev.Payload, &value)
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

	hc, err := testServer.AddConnectClient(device1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := internal.NewOWServerBinding(owsConfig, hc)
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

	hc, err := testServer.AddConnectClient(device1ID, auth.ClientTypeDevice, auth.ClientRoleDevice)
	require.NoError(t, err)
	defer hc.Disconnect()

	svc := internal.NewOWServerBinding(owsConfig, hc)
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
