package launcher_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/core/launcher/launcherclient"
	"github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var core = "mqtt"
var homeDir = "/tmp/test-launcher"
var logDir = "/tmp/test-launcher"

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

//var testClients = []msgserver.ClientAuthInfo{{
//	ClientID:   launcher.ServiceName,
//	ClientType: auth.ClientTypeService,
//	Role:       auth.ClientRoleService,
//}, {
//	ClientID:   testenv.TestAdminUserID,
//	ClientType: auth.ClientTypeUser,
//	Role:       auth.ClientRoleAdmin,
//}}

func StartService() (l launcher.ILauncher, stopFn func()) {
	const launcherID = launcher.ServiceName + "-test"
	const adminID = "admin"

	hc1, err := testServer.AddConnectClient(launcherID, auth.ClientTypeService, auth.ClientRoleService)
	if err != nil {
		panic(err)
	}
	var launcherConfig = config.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogPlugins = true
	var env = utils.GetAppEnvironment(homeDir, false)
	env.PluginsDir = "/bin" // for /bin/yes
	env.LogsDir = logDir
	env.CertsDir = homeDir

	svc := service.NewLauncherService(env, launcherConfig, hc1)
	err = svc.Start()
	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}
	//--- connect the launcher user
	hc2, err := testServer.AddConnectClient(adminID, auth.ClientTypeUser, auth.ClientRoleAdmin)
	cl := launcherclient.NewLauncherClient(launcherID, hc2)
	return cl, func() {
		hc2.Disconnect()
		_ = svc.Stop()
		hc1.Disconnect()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	var err error
	_ = os.RemoveAll(homeDir)
	_ = os.MkdirAll(homeDir, 0700)

	// include test clients
	testServer, err = testenv.StartTestServer(core)
	if err != nil {
		panic(err)
	}
	serverURL, _, _ = testServer.MsgServer.GetServerURLs()
	_ = testServer.StartAuth()
	res := m.Run()
	testServer.Stop()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	svc, cancelFunc := StartService()
	assert.NotNil(t, svc)
	time.Sleep(time.Millisecond)
	cancelFunc()
}

func TestList(t *testing.T) {
	svc, cancelFunc := StartService()
	defer cancelFunc()
	require.NotNil(t, svc)
	info, err := svc.List(false)
	assert.NoError(t, err)
	assert.NotNil(t, info)
}

func TestStartYes(t *testing.T) {
	// remove logfile from previous run
	logFile := path.Join(logDir, "yes.log")
	_ = os.Remove(logFile)

	//
	svc, cancelFunc := StartService()
	defer cancelFunc()

	assert.NotNil(t, svc)
	info, err := svc.StartPlugin("yes")
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.PID > 0)
	assert.True(t, info.StartTime != "")
	assert.FileExists(t, logFile)

	time.Sleep(time.Millisecond * 1)

	info2, err := svc.StopPlugin("yes")
	time.Sleep(time.Millisecond * 10)
	assert.NoError(t, err)
	assert.False(t, info2.Running)
	assert.True(t, info2.StopTime != "")
}

func TestStartBadName(t *testing.T) {
	svc, cancelFunc := StartService()
	defer cancelFunc()
	assert.NotNil(t, svc)

	_, err := svc.StartPlugin("notaservicename")
	require.Error(t, err)
	//
	_, err = svc.StopPlugin("notaservicename")
	require.Error(t, err)
}

func TestStartStopTwice(t *testing.T) {
	svc, cancelFunc := StartService()
	defer cancelFunc()
	assert.NotNil(t, svc)

	info, err := svc.StartPlugin("yes")
	assert.NoError(t, err)
	// second start will just return
	info2, err := svc.StartPlugin("yes")
	assert.NoError(t, err)
	_ = info2
	//assert.Equal(t, info.PID, info2.PID)

	// stop twice
	info3, err := svc.StopPlugin("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.PID, info3.PID)
	// stopping is idempotent
	info4, err := svc.StopPlugin("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.PID, info4.PID)
}

func TestStartStopAll(t *testing.T) {
	svc, cancelFunc := StartService()
	defer cancelFunc()
	assert.NotNil(t, svc)

	_, err := svc.StartPlugin("yes")
	assert.NoError(t, err)

	// result should be 1 service running
	info, err := svc.List(true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(info))

	// stopping
	err = svc.StopAllPlugins()
	assert.NoError(t, err)

	// result should be no service running
	info, err = svc.List(true)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(info))

}
