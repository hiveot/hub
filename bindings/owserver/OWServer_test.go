package owserver_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
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

// TODO: switch for testing with real owserver

var tempFolder string
var owsConfig config.OWServerConfig
var owsSimulationFile string // simulation file
var ts *testenv.TestServer

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	var err error
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "./docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	// uncomment the next line to discover and test with a real owserver
	//owsSimulationFile = ""
	logging.SetLogging("info", "")

	owsConfig = *config.NewConfig()
	owsConfig.OWServerURL = owsSimulationFile
	//
	ts = testenv.StartTestServer(true)
	if err != nil {
		panic("unable to start test server: " + err.Error())
	}
	result := m.Run()
	time.Sleep(time.Millisecond * 10)
	println("stopping testserver")

	ts.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")
	const device1ID = "device1"

	hc, _ := ts.AddConnectAgent(device1ID)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	assert.NoError(t, err)
	defer svc.Stop()
	//time.Sleep(time.Second)
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const device1ID = "device1"

	t.Log("--- TestPoll ---")
	hc, _ := ts.AddConnectAgent(device1ID)
	defer hc.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)

	// Count the number of received TD events
	err := hc.Subscribe("", "")
	require.NoError(t, err)
	hc.SetEventHandler(func(ev *things.ThingMessage) error {
		slog.Info("received event", "id", ev.Key)
		if ev.Key == vocab.EventTypeProperties {
			var value map[string]interface{}
			err2 := ev.Unmarshal(&value)
			assert.NoError(t, err2)
		} else {
			var value interface{}
			err2 := ev.Unmarshal(&value)
			assert.NoError(t, err2)
		}
		tdCount.Add(1)
		return nil
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	err = svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// wait until startup poll completed
	time.Sleep(time.Millisecond * 200)

	// the simulation file contains 3 things. The service is 1 Thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Log("--- TestPollInvalidEDSAddress ---")
	const device1ID = "device1"

	hc, _ := ts.AddConnectAgent(device1ID)
	defer hc.Disconnect()

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc := service.NewOWServerBinding(&badConfig)
	err := svc.Start(hc)
	assert.NoError(t, err)
	defer svc.Stop()

	time.Sleep(time.Millisecond * 10)

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Log("--- TestAction ---")
	const agentID = "agent-1"
	const user1ID = "operator1"
	// node in test data
	const thingID = "C100100000267C7E"
	var dThingID = things.MakeDigiTwinThingID(agentID, thingID)
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	hc, _ := ts.AddConnectAgent(agentID)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectUser(user1ID, authn.ClientRoleOperator)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.Rpc(dThingID, actionName, &actionValue, nil)
	// can't write to a simulation
	assert.Error(t, err)

	//time.Sleep(time.Second * 1) // debugging delay
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

	hc, _ := ts.AddConnectAgent(device1ID)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectUser(user1ID, authn.ClientRoleManager)
	defer hc2.Disconnect()
	dThingID := things.MakeDigiTwinThingID(device1ID, nodeID)
	err = hc2.Rpc(dThingID, configName, &configValue, nil)
	// can't write to a simulation. How to test for real?
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
