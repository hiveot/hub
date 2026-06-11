package launcher_test

import (
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	launcher "github.com/hiveot/hub/services/launcher/api"
	"github.com/hiveot/hub/services/launcher/config"
	"github.com/hiveot/hub/services/launcher/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// var homeDir = "/tmp/test-launcher"
var logDir = "/tmp/test-launcher"

// the following are set by the testmain
var testEnv *testenv.TestEnv

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

func startService() (l *consumer.Consumer, stopFn func()) {
	const launcherID = launcher.AdminAgentID
	const adminID = "admin"

	testEnv = testenv.NewTestEnv()

	//hc1, _ := ts.AddConnectService(launcherID)
	var launcherConfig = config.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogPlugins = true
	//launcherConfig.LogsDir = ts.AppEnv.LogsDir
	launcherConfig.LogsDir = logDir
	// todo: add tests for providing discovery
	launcherConfig.ProvideDirectoryURL = false
	launcherConfig.ProvideServerURL = false
	//var env = plugin.GetAppEnvironment(ts.AppEnv.HomeDir, false)

	binDir := testEnv.AppEnv.BinDir
	pluginsDir := "/bin" // for /bin/yes
	certsDir := testEnv.AppEnv.CertsDir
	clientID := launcherID

	//env.LogsDir = logDir
	//env.CertsDir = homeDir
	//env.CaCert = ts.Certs.CaCert

	serverURL := testEnv.GetServerURL(authn.ClientTypeService)
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
	co1, _, _ := testEnv.AddConnectConsumer(adminID, authn.ClientRoleAdmin)
	return co1, func() {
		co1.Disconnect()
		//hc1.Disconnect()
		_ = svc.Stop()
		time.Sleep(time.Millisecond)
		testEnv.Stop()
	}
}

func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	svc, cancelFunc := startService()
	assert.NotNil(t, svc)
	time.Sleep(time.Millisecond)
	cancelFunc()
}

func TestList(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	userID := "user1"

	co1, cancelFunc := startService()
	defer cancelFunc()
	require.NotNil(t, co1)
	// using the /bin directory yields a larger number of potential plugins
	infoList, err := launcher.AdminListPlugins(co1, false)
	require.NoError(t, err)
	assert.Greater(t, len(infoList), 10)

	co2, _, _ := testEnv.AddConnectConsumer(userID, authn.ClientRoleAdmin)
	defer co2.Disconnect()
	infoList2, err := launcher.AdminListPlugins(co2, false)
	require.NoError(t, err)
	require.NotEmpty(t, infoList2)
}

func TestListNoPermission(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	userID := "user1"

	co1, cancelFunc := startService()
	defer cancelFunc()
	require.NotNil(t, co1)

	co2, _, _ := testEnv.AddConnectConsumer(userID, authn.ClientRoleNone)
	defer co1.Disconnect()
	info2, err := launcher.AdminListPlugins(co2, false)
	require.Error(t, err, "user without role should not be able to use launcher")
	require.Empty(t, info2)
}

func TestStartYes(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	// remove logfile from previous run
	logFile := path.Join(logDir, "yes.log")
	_ = os.Remove(logFile)

	//
	co1, cancelFunc := startService()
	defer cancelFunc()

	assert.NotNil(t, co1)
	info, err := launcher.AdminStartPlugin(co1, "yes")
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.Pid > 0)
	assert.True(t, info.StartedTime != "")
	assert.FileExists(t, logFile)

	time.Sleep(time.Millisecond * 1)

	info2, err := launcher.AdminStopPlugin(co1, "yes")
	time.Sleep(time.Millisecond * 10)
	assert.NoError(t, err)
	assert.False(t, info2.Running)
	assert.True(t, info2.StoppedTime != "")
}

func TestStartBadName(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	co1, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, co1)

	// FIXME: the error is not received - how to return an error in an action response?!

	_, err := launcher.AdminStartPlugin(co1, "notaservicename")
	require.Error(t, err)
	//
	_, err = launcher.AdminStopPlugin(co1, "notaservicename")
	require.Error(t, err)
}

func TestStartStopTwice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	co1, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, co1)

	info, err := launcher.AdminStartPlugin(co1, "yes")
	assert.NoError(t, err)
	// second start will just return
	info2, err := launcher.AdminStartPlugin(co1, "yes")
	assert.NoError(t, err)
	_ = info2
	//assert.Equal(t, info.PID, info2.PID)

	// stop twice
	info3, err := launcher.AdminStopPlugin(co1, "yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.Pid, info3.Pid)
	// stopping is idempotent
	info4, err := launcher.AdminStopPlugin(co1, "yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.Pid, info4.Pid)
}

func TestStartStopAll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	co1, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, co1)

	_, err := launcher.AdminStartPlugin(co1, "yes")
	assert.NoError(t, err)

	// result should be 1 service running
	pluginList, err := launcher.AdminListPlugins(co1, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pluginList))

	// stopping
	err = launcher.AdminStopAllPlugins(co1, false)
	assert.NoError(t, err)

	// result should be no service running
	pluginList, err = launcher.AdminListPlugins(co1, true)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pluginList))
	assert.NoError(t, err)
}
