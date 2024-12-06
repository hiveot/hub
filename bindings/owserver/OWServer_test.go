package owserver_test

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/owserver/config"
	"github.com/hiveot/hub/bindings/owserver/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/wot/transports/utils"
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

const agentUsesWSS = false
const agentID = "owserver"
const device1ID = "2A000003BB170B28" // <-- from the simulation file

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
	t.Log("--- TestStartStop (without state service) ---")

	svc := service.NewOWServerBinding(&owsConfig)

	hc, _ := ts.AddConnectAgent(agentID, agentUsesWSS)
	connected, _, _ := hc.GetConnectionStatus()
	require.Equal(t, true, connected)
	defer hc.Disconnect()

	err := svc.Start(hc)
	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32
	const userID = "user1"

	t.Log("--- TestPoll (without state service)  ---")
	ag1, _ := ts.AddConnectAgent(agentID, agentUsesWSS)
	defer ag1.Disconnect()
	cl1, _ := ts.AddConnectConsumer(userID, authz.ClientRoleManager)
	defer cl1.Disconnect()
	svc := service.NewOWServerBinding(&owsConfig)

	// Count the number of received TD events
	err := cl1.ObserveProperty("", "")
	err = cl1.Subscribe("", "")
	require.NoError(t, err)
	cl1.SetNotificationHandler(func(msg *transports.ThingMessage) {
		slog.Info("received message", "MessageType", msg.Operation, "id", msg.Name)
		var value interface{}
		err2 := utils.DecodeAsObject(msg.Data, &value)
		assert.NoError(t, err2)

		tdCount.Add(1)
		return
	})
	assert.NoError(t, err)

	// start the service which publishes TDs
	err = svc.Start(ag1)
	require.NoError(t, err)

	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	// the simulation file contains 3 things. The service is 1 Thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))

	// get events from the digitwin
	dThingID := td.MakeDigiTwinThingID(agentID, device1ID)
	events, err := digitwin.ValuesReadAllEvents(cl1, dThingID)
	require.NoError(t, err)
	// only 1 event (temperature) is expected
	require.True(t, len(events) == 1)
}

func TestPollInvalidEDSAddress(t *testing.T) {
	t.Log("--- TestPollInvalidEDSAddress ---")

	hc, _ := ts.AddConnectAgent(agentID, agentUsesWSS)
	defer hc.Disconnect()

	badConfig := owsConfig // copy
	badConfig.OWServerURL = "http://invalidAddress/"
	svc := service.NewOWServerBinding(&badConfig)

	err := svc.Start(hc)
	assert.NoError(t, err)
	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	_, err = svc.PollNodes()
	assert.Error(t, err)
}

func TestAction(t *testing.T) {
	t.Log("--- TestAction (without state service)  ---")
	const user1ID = "operator1"
	// node in test data
	var dThingID = td.MakeDigiTwinThingID(agentID, device1ID)
	var actionName = "RelayFunction" // the action attribute as defined by the device
	var actionValue = "1"

	hc, _ := ts.AddConnectAgent(agentID, agentUsesWSS)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectConsumer(user1ID, authz.ClientRoleOperator)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.SendRequest(dThingID, actionName, &actionValue, nil)
	// can't write to a simulation
	assert.Error(t, err)

	//time.Sleep(time.Second * 1) // debugging delay
}

func TestConfig(t *testing.T) {
	t.Log("--- TestConfig (without state service)  ---")
	const user1ID = "manager1"
	var configName = "LEDFunction"
	var configValue = ([]byte)("1")

	hc, _ := ts.AddConnectAgent(agentID, agentUsesWSS)
	defer hc.Disconnect()

	svc := service.NewOWServerBinding(&owsConfig)
	err := svc.Start(hc)
	require.NoError(t, err)
	defer svc.Stop()

	// give heartbeat time to run
	time.Sleep(time.Millisecond * 10)

	// note that the simulation file doesn't support writes so this logs an error
	hc2, _ := ts.AddConnectConsumer(user1ID, authz.ClientRoleManager)
	defer hc2.Disconnect()
	dThingID := td.MakeDigiTwinThingID(agentID, device1ID)
	err = hc2.SendRequest(dThingID, configName, &configValue, nil)
	// can't write to a simulation. How to test for real?
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
