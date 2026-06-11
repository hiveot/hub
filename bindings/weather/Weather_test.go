package weather_test

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/weather/config"
	"github.com/hiveot/hub/bindings/weather/providers"
	"github.com/hiveot/hub/bindings/weather/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLocation1 = config.WeatherLocation{
	ID:             "Vancouver-1",
	Name:           "Vancouver",
	Latitude:       "49.286",
	Longitude:      "-123.182",
	CurrentEnabled: true,
	HourlyEnabled:  true,
}
var testLocation2 = config.WeatherLocation{
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
var weatherConfig = config.NewWeatherConfig()

func Setup() (testEnv *testenv.TestEnv, ag *agent.Agent, stopFn func()) {
	var cancelFn func()

	utils.SetLogging("warn", "")
	testEnv, cancelFn = testenv.StartTestEnv("")

	ag, _, _ = testEnv.NewRCAgent(agentID, nil)
	utils.SetLogging("info", "")

	return testEnv, ag, func() {
		utils.SetLogging("warn", "")
		ag.Stop()
		cancelFn()
	}
}

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

	_, ag, stopFn := Setup()
	defer stopFn()

	svc := service.NewWeatherBinding(storePath, weatherConfig)
	err := svc.Start()

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
	_, ag, stopFn := Setup()
	defer stopFn()

	svc := service.NewWeatherBinding(storePath, weatherConfig)
	err := svc.Start()

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
	client1ID := "client1"

	testEnv, ag, stopFn := Setup()
	defer stopFn()

	svc := service.NewWeatherBinding(storePath, weatherConfig)
	err := svc.Start()

	require.NoError(t, err)
	err = svc.AddLocation(testLocation1)
	err = svc.AddLocation(testLocation2)
	require.NoError(t, err)

	require.NoError(t, err)
	// give heartbeat time to run
	time.Sleep(time.Millisecond * 1)

	co1, cc1, _ := testEnv.NewConnectedConsumer(client1ID, authn.ClientRoleAdmin, false)
	defer co1.Stop()
	defer cc1.Stop()

	thingID := testLocation1.ID
	err = co1.WriteProperty(thingID, service.PropNameCurrentEnabled, false, true)
	require.NoError(t, err)

	loc1, found := svc.LocationStore().Get(testLocation1.ID)
	require.True(t, found)
	require.Equal(t, testLocation1.ID, loc1.ID)
	require.Equal(t, false, loc1.CurrentEnabled)

	err = svc.Poll()
	assert.NoError(t, err)

	svc.Stop()
}
