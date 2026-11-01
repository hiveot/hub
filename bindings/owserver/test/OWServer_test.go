package owserver_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/consumer"
	directory_client "github.com/hiveot/hivekit/go/cells/directory/client"
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

const agentID = "owserver"

// const device1ID = "2A000003BB170B28" // <-- from the simulation file
const device1ID = "C100100000267C7E" // <-- from the simulation file

// TestMain creates the test environment.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	cwd, _ := os.Getwd()
	testFolder := path.Join(cwd, "./")
	owsSimulationFile = "file://" + path.Join(testFolder, "owserver-simulation.xml")
	// uncomment the next line to discover and test with a real owserver
	//owsSimulationFile = ""
	utils.SetLogging("info", "")

	owsConfig = *owserver.NewOWServerConfig()
	owsConfig.OWServerURL = owsSimulationFile
	testEnv = testenv.NewTestEnv(true)
	storePath = testEnv.Env.GetStorageDir(owserver.DefaultOWServerThingID)

	result := m.Run()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	defer svc.Stop()
	require.NoError(t, err)
	co1 := consumer.NewConsumer(svc, nil)
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
	// var tdCount atomic.Int32
	const userID = "user1"

	// request chain: consumer -> owserver -> directoryclient
	co1 := consumer.NewConsumer(nil, nil)
	svc, err := internal.NewOWServerBinding("", storePath, &owsConfig)
	dirCl := directory_client.NewDirectoryClient(nil, nil)

	svc.SetRequestSink(dirCl)
	co1.SetRequestSink(svc)
	defer co1.Stop()

	svc.Start()
	// give heartbeat a chance to run. stop will wait for it to complete
	time.Sleep(time.Millisecond * 100)
	svc.Stop()

	tdList := dirCl.Cache().GetAllThings(0, 0)
	// the simulation file contains 3 things. The service itself is also a Thing.
	assert.GreaterOrEqual(t, len(tdList), 4)

	// get events that were published
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
	svc.Start()
	defer svc.Stop()

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
	co1, cc2, _ := testEnv.NewConnectedConsumer(user1ID, api.ClientRoleOperator)
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
	co1, cc2, _ := testEnv.NewConnectedConsumer(user1ID, api.ClientRoleManager)
	defer cc2.Close()
	// dThingID := digitwin.MakeDigitwinID(agentID, device1ID)
	err = co1.WriteProperty(device1ID, configName, &configValue, true)

	// can't write to a simulation file. Write should fail.
	assert.Error(t, err)

	//time.Sleep(time.Second*10)  // debugging delay
}
