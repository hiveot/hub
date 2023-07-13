package internal_test

import (
	"encoding/json"
	"github.com/hiveot/hub/lib/testenv"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/lib/logging"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/plugins/owserver/internal"
)

// var homeFolder string
var testUrl = "" // client connect url

var testCerts testenv.TestAuthBundle

var tempFolder string
var owsConfig internal.OWServerConfig
var owsSimulationFile string // simulation file

// launch the hub
func startServer() (svc *testenv.TestServer) {
	svc = testenv.NewTestServer(testCerts.ServerCert, testCerts.CaCert)
	clientURL, err := svc.Start()
	testUrl = clientURL
	if err != nil {
		panic("unable to start test server")
	}
	return svc
}

// TestMain run mosquitto and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	testCerts = testenv.CreateTestAuthBundle()
	tempFolder = path.Join(os.TempDir(), "test-owserver")
	cwd, _ := os.Getwd()
	homeFolder := path.Join(cwd, "../docs")
	owsSimulationFile = "file://" + path.Join(homeFolder, "owserver-simulation.xml")
	logging.SetLogging("info", "")

	owsConfig = internal.NewConfig()
	owsConfig.ID = testCerts.DeviceID
	owsConfig.OWServerAddress = owsSimulationFile

	//
	srv := startServer()

	result := m.Run()
	time.Sleep(time.Second)

	srv.Stop()
	if result == 0 {
		_ = os.RemoveAll(tempFolder)
	}

	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	slog.Info("--- TestStartStop ---")

	hc := hubconn.NewHubClient(owsConfig.ID)
	err := hc.ConnectWithCert(testUrl, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	require.NoError(t, err)
	svc := internal.NewOWServerBinding(owsConfig, hc)
	go func() {
		err := svc.Start()
		assert.NoError(t, err)
	}()
	time.Sleep(time.Second)
	svc.Stop()
}

func TestPoll(t *testing.T) {
	var tdCount atomic.Int32

	slog.Info("--- TestPoll ---")
	hc := hubconn.NewHubClient(owsConfig.ID)
	err := hc.ConnectWithCert(testUrl, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	require.NoError(t, err)
	svc := internal.NewOWServerBinding(owsConfig, hc)

	// Count the number of received TD events
	err = hc.SubEvent("", "", vocab.EventNameTD,
		func(ev *thing.ThingValue) {
			if ev.ID == vocab.EventNameProps {
				var value map[string][]byte
				err2 := json.Unmarshal(ev.Data, &value)
				assert.NoError(t, err2)
				//for propName, propValue := range value {
				//	pv := string(propValue)
				//slog.Infof("%s: %s", propName, pv)
				//}
			} else {
				var value interface{}
				err2 := json.Unmarshal(ev.Data, &value)
				assert.NoError(t, err2)
			}
			tdCount.Add(1)
		})
	assert.NoError(t, err)

	// start the service which publishes TDs
	go func() {
		err := svc.Start()
		assert.NoError(t, err)
	}()

	// wait until startup poll completed
	time.Sleep(time.Millisecond * 1000)

	// the simulation file contains 3 things. The service is 1 thing.
	assert.GreaterOrEqual(t, tdCount.Load(), int32(4))
	svc.Stop()
}

func TestPollInvalidEDSAddress(t *testing.T) {
	slog.Info("--- TestPollInvalidEDSAddress ---")

	hc := hubconn.NewHubClient(owsConfig.ID)
	err := hc.ConnectWithCert(testUrl, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	require.NoError(t, err)
	svc := internal.NewOWServerBinding(owsConfig, hc)
	svc.Config.OWServerAddress = "http://invalidAddress/"

	go func() {
		err := svc.Start()
		assert.NoError(t, err)
	}()

	time.Sleep(time.Millisecond * 10)

	_, err = svc.PollNodes()
	assert.Error(t, err)
	svc.Stop()
}

func TestAction(t *testing.T) {
	slog.Info("--- TestAction ---")
	// node in test data
	const nodeID = "C100100000267C7E"
	//var nodeAddr = thing.MakeThingAddr(owsConfig.ID, nodeID)
	var actionName = vocab.VocabRelay
	var actionValue = ([]byte)("1")

	hc := hubconn.NewHubClient(owsConfig.ID)
	err := hc.ConnectWithCert(testUrl, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	require.NoError(t, err)

	svc := internal.NewOWServerBinding(owsConfig, hc)
	go func() {
		err := svc.Start()
		assert.NoError(t, err)
	}()
	// give Start time to run
	time.Sleep(time.Millisecond * 10)

	// This will log an error as the simulation file doesn't accept writes
	err = hc.PubAction(owsConfig.ID, nodeID, actionName, actionValue)
	assert.NoError(t, err)

	time.Sleep(time.Second * 3)
	svc.Stop()
}
