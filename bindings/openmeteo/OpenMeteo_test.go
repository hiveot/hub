package openmeteo_test

import (
	"fmt"
	"github.com/hiveot/hub/bindings/openmeteo/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

var storePath string
var tempFolder string
var ts *testenv.TestServer

var testLocation1 = service.WeatherConfiguration{
	ID:             "Vancouver-1",
	LocationName:   "Vancouver",
	Latitude:       "49.286",
	Longitude:      "-123.182",
	CurrentWeather: true,
	DailyForecast:  true,
}
var testLocation2 = service.WeatherConfiguration{
	ID:             "SalmonArm-1",
	LocationName:   "Salmon Arm, BC",
	Latitude:       "50.690043",
	Longitude:      "-119.299657",
	CurrentWeather: true,
	DailyForecast:  true,
}

const agentID = "openmeteo"
const clientID = "user1"

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	tempFolder = path.Join(os.TempDir(), "test-openmeteo")
	storePath = path.Join(tempFolder, "openmeteo-config")

	logging.SetLogging("info", "")
	ts = testenv.StartTestServer(true)

	result := m.Run()

	time.Sleep(time.Millisecond * 10)
	println("stopping testserver")
	ts.Stop()
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop (without state service) ---")

	svc := service.NewOpenMeteoBinding(storePath)

	ag, _, _ := ts.AddConnectAgent(agentID)
	//connected := ag.IsConnected()
	//require.Equal(t, true, connected)
	defer ag.Disconnect()

	err := svc.Start(ag)
	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)
	svc.Stop()
}

func TestPoll(t *testing.T) {

	t.Log("--- TestPoll   ---")
	svc := service.NewOpenMeteoBinding(storePath)
	ag, _, _ := ts.AddConnectAgent(agentID)
	defer ag.Disconnect()

	err := svc.Start(ag)
	require.NoError(t, err)
	err = svc.Locations().Add(&testLocation2)
	require.NoError(t, err)

	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)

	// the simulation file contains the open-meteo response
	// todo
	t1 := time.Now()

	err = svc.Poll()
	assert.NoError(t, err)
	t2 := time.Now()
	duration := t2.Sub(t1)
	fmt.Printf("Duration: %d msec\n", duration.Milliseconds())

	svc.Stop()
}
