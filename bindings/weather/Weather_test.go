package weather_test

import (
	"fmt"
	"github.com/hiveot/hub/bindings/weather/config"
	providers2 "github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/bindings/weather/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

var testLocation1 = providers2.WeatherLocationConfig{
	ID:              "Vancouver-1",
	LocationName:    "Vancouver",
	Latitude:        "49.286",
	Longitude:       "-123.182",
	CurrentEnabled:  true,
	ForecastEnabled: true,
}
var testLocation2 = providers2.WeatherLocationConfig{
	ID:              "Amsterdam-1",
	LocationName:    "Amsterdam, NL",
	Latitude:        "52.375009",
	Longitude:       "4.895107",
	CurrentEnabled:  true,
	ForecastEnabled: true,
}

const agentID = "weather"
const clientID = "user1"

var storePath string
var tempFolder string
var weatherConfig = config.NewWeatherConfig()

func Setup() (ts *testenv.TestServer, ag *messaging.Agent, stopFn func()) {
	logging.SetLogging("warn", "")
	ts = testenv.StartTestServer(true)

	ag, _, _ = ts.AddConnectAgent(agentID)
	logging.SetLogging("info", "")

	return ts, ag, func() {
		logging.SetLogging("warn", "")
		ag.Disconnect()
		ts.Stop()
	}
}

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup environment
	tempFolder = path.Join(os.TempDir(), "test-openmeteo")
	storePath = path.Join(tempFolder, "openmeteo-config")

	logging.SetLogging("info", "")
	//ts = testenv.StartTestServer(true)

	result := m.Run()

	time.Sleep(time.Millisecond * 10)
	//println("stopping testserver")
	//ts.Stop()
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	_, ag, stopFn := Setup()
	defer stopFn()

	svc := service.NewWeatherBinding(storePath, weatherConfig)

	err := svc.Start(ag)
	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)

	svc.Stop()
}

func TestPollDirect(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	t1 := time.Now()
	meteo := providers2.NewOpenMeteoProvider()
	current, err := meteo.ReadCurrent(testLocation1)
	require.NoError(t, err)
	t2 := time.Now()
	duration := t2.Sub(t1)
	fmt.Printf("Duration: %d msec\n", duration.Milliseconds())
	assert.NotEmpty(t, current.CloudCover)
	assert.NotEmpty(t, current.Humidity)
	assert.NotEmpty(t, current.Rain)
	assert.NotEmpty(t, current.Temperature)
	assert.NotEmpty(t, current.WindDirection)
	assert.NotEmpty(t, current.WindSpeed)
}

func TestPollFromService(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	_, ag, stopFn := Setup()
	defer stopFn()

	svc := service.NewWeatherBinding(storePath, weatherConfig)
	err := svc.Start(ag)
	require.NoError(t, err)
	err = svc.LocationStore().Add(&testLocation1)
	err = svc.LocationStore().Add(&testLocation2)
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
