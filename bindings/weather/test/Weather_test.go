package weather_test

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/weather"
	"github.com/hiveot/hub/bindings/weather/internal"
	weather_pkg "github.com/hiveot/hub/bindings/weather/pkg"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLocation1 = weather.WeatherLocation{
	ID:             "Vancouver-1",
	Name:           "Vancouver",
	Latitude:       "49.286",
	Longitude:      "-123.182",
	CurrentEnabled: true,
	HourlyEnabled:  true,
}
var testLocation2 = weather.WeatherLocation{
	ID:             "Amsterdam-1",
	Name:           "Amsterdam, NL",
	Latitude:       "52.375009",
	Longitude:      "4.895107",
	CurrentEnabled: true,
	HourlyEnabled:  true,
}

const agentID = "weather"
const clientID = "user1"

var storePath string
var tempFolder string
var weatherConfig = weather.NewWeatherConfig()

// TestMain run test server and use the project test folder as the home folder.
// All tests are run using the simulation file.
func TestMain(m *testing.M) {
	// setup a clean environment
	tempFolder = path.Join(os.TempDir(), "test-openmeteo")
	storePath = path.Join(tempFolder, "openmeteo-config")
	_ = os.RemoveAll(storePath)

	utils.SetLogging("info", "")
	//ts = testenv.StartTestServer(true)

	result := m.Run()

	time.Sleep(time.Millisecond * 10)
	//println("stopping testserver")
	//ts.Stop()
	os.Exit(result)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := weather_pkg.NewWeatherService("", storePath, weatherConfig)
	require.NoError(t, err)

	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)

	svc.Stop()
}

func TestPollDirect(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	t1 := time.Now()
	meteo := providers.NewOpenMeteoProvider()
	current, err := meteo.ReadCurrent(testLocation1)
	require.NoError(t, err)
	t2 := time.Now()
	duration := t2.Sub(t1)
	fmt.Printf("Duration: %d msec\n", duration.Milliseconds())
	assert.NotEmpty(t, current.CloudCover)
	assert.NotEmpty(t, current.Humidity)
	assert.NotEmpty(t, current.Rain)
	assert.NotEmpty(t, current.Temperature)
	assert.NotEmpty(t, current.WindHeading)
	assert.NotEmpty(t, current.WindGusts)
	assert.NotEmpty(t, current.WindSpeed)
}

func TestPollFromService(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	testEnv := testenv.NewTestEnv(true)
	storePath := testEnv.Env.GetStorageDir("weather")

	svc, err := internal.NewWeatherService("", storePath, weatherConfig)
	require.NoError(t, err)

	err = svc.LocationStore().Add(testLocation1)
	err = svc.LocationStore().Add(testLocation2)
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

func TestDisableCurrent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := internal.NewWeatherService("", storePath, weatherConfig)
	require.NoError(t, err)

	err = svc.AddLocation(testLocation1)
	err = svc.AddLocation(testLocation2)
	require.NoError(t, err)

	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)

	co1 := consumer.NewConsumer(svc, nil)
	defer co1.Stop()

	thingID := testLocation1.ID
	err = co1.WriteProperty(thingID, internal.PropNameCurrentEnabled, false, true)
	require.NoError(t, err)

	loc1, found := svc.LocationStore().Get(testLocation1.ID)
	require.True(t, found)
	require.Equal(t, testLocation1.ID, loc1.ID)
	require.Equal(t, false, loc1.CurrentEnabled)

	err = svc.Poll()
	assert.NoError(t, err)

	svc.Stop()
}
