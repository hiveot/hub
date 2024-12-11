package launcher_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/services/launcher/config"
	"github.com/hiveot/hub/services/launcher/launcherapi"
	"github.com/hiveot/hub/services/launcher/launcherclient"
	"github.com/hiveot/hub/services/launcher/service"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

// var homeDir = "/tmp/test-launcher"
var logDir = "/tmp/test-launcher"

// the following are set by the testmain
var testServer *testenv.TestServer

const agentUsesWSS = false

//var testClients = []msgserver.ClientAuthInfo{{
//	SenderID:   launcher.ServiceName,
//	ClientType: auth.ClientTypeService,
//	Role:       auth.ClientRoleService,
//}, {
//	SenderID:   testenv.TestAdminUserID,
//	ClientType: auth.ClientTypeUser,
//	Role:       auth.ClientRoleAdmin,
//}}

func startService() (l *launcherclient.LauncherClient, stopFn func()) {
	const launcherID = launcherapi.AgentID
	const adminID = "admin"

	testServer = testenv.StartTestServer(true)

	//hc1, _ := testServer.AddConnectService(launcherID)
	var launcherConfig = config.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogPlugins = true
	//launcherConfig.LogsDir = testServer.AppEnv.LogsDir
	launcherConfig.LogsDir = logDir
	//var env = plugin.GetAppEnvironment(testServer.AppEnv.HomeDir, false)

	binDir := testServer.AppEnv.BinDir
	pluginsDir := "/bin" // for /bin/yes
	certsDir := testServer.AppEnv.CertsDir
	clientID := launcherapi.AgentID

	//env.LogsDir = logDir
	//env.CertsDir = homeDir
	//env.CaCert = testServer.Certs.CaCert

	serverURL := testServer.GetServerURL(authn.ClientTypeService)
	svc := service.NewLauncherService(
		serverURL, clientID, binDir, pluginsDir, certsDir, launcherConfig)
	err := svc.Start()
	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}
	// the agent receives actions to execute on the service
	//agent := service.StartLauncherAgent(svc, hc1)
	//_ = agent
	//--- connect the launcher user
	hc2, _ := testServer.AddConnectConsumer(adminID, authz.ClientRoleAdmin)
	cl := launcherclient.NewLauncherClient(launcherID, hc2)
	return cl, func() {
		hc2.Disconnect()
		//hc1.Disconnect()
		_ = svc.Stop()
		time.Sleep(time.Millisecond)
		testServer.Stop()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	var err error
	if err != nil {
		panic(err)
	}
	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())
	svc, cancelFunc := startService()
	assert.NotNil(t, svc)
	time.Sleep(time.Millisecond)
	cancelFunc()
}

func TestList(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())
	userID := "user1"

	svc, cancelFunc := startService()
	defer cancelFunc()
	require.NotNil(t, svc)
	// using the /bin directory yields a larger number of potential plugins
	info, err := svc.List(false)
	require.NoError(t, err)
	assert.Greater(t, len(info), 10)

	hc, _ := testServer.AddConnectConsumer(userID, authz.ClientRoleAdmin)
	defer hc.Disconnect()
	cl := launcherclient.NewLauncherClient("", hc)
	info2, err := cl.List(false)
	require.NoError(t, err)
	require.NotEmpty(t, info2)
}

func TestListNoPermission(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())
	userID := "user1"

	svc, cancelFunc := startService()
	defer cancelFunc()
	require.NotNil(t, svc)

	hc, _ := testServer.AddConnectConsumer(userID, authz.ClientRoleNone)
	defer hc.Disconnect()
	cl := launcherclient.NewLauncherClient("", hc)
	info2, err := cl.List(false)
	require.Error(t, err, "user without role should not be able to use launcher")
	require.Empty(t, info2)
}

func TestStartYes(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())
	// remove logfile from previous run
	logFile := path.Join(logDir, "yes.log")
	_ = os.Remove(logFile)

	//
	svc, cancelFunc := startService()
	defer cancelFunc()

	assert.NotNil(t, svc)
	info, err := svc.StartPlugin("yes")
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.PID > 0)
	assert.True(t, info.StartTimeMSE != 0)
	assert.FileExists(t, logFile)

	time.Sleep(time.Millisecond * 1)

	info2, err := svc.StopPlugin("yes")
	time.Sleep(time.Millisecond * 10)
	assert.NoError(t, err)
	assert.False(t, info2.Running)
	assert.True(t, info2.StopTimeMSE != 0)
}

func TestStartBadName(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())

	svc, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, svc)

	_, err := svc.StartPlugin("notaservicename")
	require.Error(t, err)
	//
	_, err = svc.StopPlugin("notaservicename")
	require.Error(t, err)
}

func TestStartStopTwice(t *testing.T) {
	fmt.Printf("---%s---\n", t.Name())
	svc, cancelFunc := startService()
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
	fmt.Printf("---%s---\n", t.Name())
	svc, cancelFunc := startService()
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
	svc.Stop()
}
