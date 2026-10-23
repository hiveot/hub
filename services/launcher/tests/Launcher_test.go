package launcher_test

import (
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var homeDir = "/tmp/hiveot/launcher-test"

// var logDir = "/tmp/hiveot/launcher-test"

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
	const launcherID = launcher.LauncherServiceID
	const adminID = "admin"

	appEnv := api.NewHiveEnvironment(homeDir, false)
	appEnv.LogsDir = appEnv.HomeDir
	appEnv.PluginsDir = appEnv.HomeDir

	os.MkdirAll(appEnv.HomeDir, 0750)

	testEnv = testenv.NewTestEnv(true)
	srv := testEnv.StartTestServer("")
	err := srv.Start()

	// authn is needed to create auth tokens
	authnConfig := authn.NewAuthnConfig(appEnv.CertsDir, homeDir)
	authnSvc := authn_service.NewAuthnService(authnConfig)
	// directory is needed to write TD (optional - WriteTD)
	dirSvc := directory_service.NewDirectoryService("", "", nil, nil)

	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}
	//hc1, _ := ts.AddConnectService(launcherID)
	var launcherConfig = launcher.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogPlugins = true

	// todo: add tests for providing discovery
	launcherConfig.ProvideDirectoryURL = false
	launcherConfig.ProvideServerURL = false
	launcherConfig.WriteTD = false

	// launcher uses authn to create credentials and dirSvc to write TD
	svc := launcher_service.NewLauncherService(authnSvc, appEnv, launcherConfig)
	authnSvc.SetRequestSink(dirSvc)

	err = svc.Start()
	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}
	// the agent receives actions to execute on the service
	//agent := service.StartLauncherAgent(svc, hc1)
	//_ = agent

	//--- connect the launcher user
	co1, co1cc, _ := testEnv.NewTestConsumer(adminID, api.ClientRoleAdmin, false)
	_ = co1cc.Connect()
	return co1, func() {
		co1cc.Close()
		time.Sleep(time.Millisecond)
		svc.Stop()
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
	launcherCl := launcher_client.NewLauncherClient("", co1)
	infoList, err := launcherCl.ListPlugins(false)
	require.NoError(t, err)
	assert.Greater(t, len(infoList), 10)

	cc2, _ := testEnv.NewTestClient(userID, api.ClientRoleAdmin)
	_ = cc2.Connect()
	defer cc2.Close()
	infoList2, err := launcherCl.ListPlugins(false)
	require.NoError(t, err)
	require.NotEmpty(t, infoList2)
}

func TestListNoPermission(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	userID := "user1"

	co1, cancelFunc := startService()
	defer cancelFunc()
	require.NotNil(t, co1)

	cc2, _ := testEnv.NewTestClient(userID, api.ClientRoleNone)
	cc2.Connect()
	defer cc2.Close()
	launcherCl := launcher_client.NewLauncherClient("", cc2)
	info2, err := launcherCl.ListPlugins(false)
	require.Error(t, err, "user without role should not be able to use launcher")
	require.Empty(t, info2)
}

func TestStartYes(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	// remove logfile from previous run
	logFile := path.Join(homeDir, "yes.log")
	_ = os.Remove(logFile)

	//
	co1, cancelFunc := startService()
	defer cancelFunc()

	assert.NotNil(t, co1)
	launcherCl := launcher_client.NewLauncherClient("", co1)
	info, err := launcherCl.StartPlugin("yes")
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.Pid > 0)
	assert.True(t, info.StartedTime != "")
	assert.FileExists(t, logFile)

	time.Sleep(time.Millisecond * 1)

	info2, err := launcherCl.StopPlugin("yes")
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

	launcherCl := launcher_client.NewLauncherClient("", co1)
	_, err := launcherCl.StartPlugin("notaservicename")
	require.Error(t, err)
	//
	_, err = launcherCl.StopPlugin("notaservicename")
	require.Error(t, err)
}

func TestStartStopTwice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	co1, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, co1)

	launcherCl := launcher_client.NewLauncherClient("", co1)

	info, err := launcherCl.StartPlugin("yes")
	assert.NoError(t, err)
	// second start will just return
	info2, err := launcherCl.StartPlugin("yes")
	assert.NoError(t, err)
	_ = info2
	//assert.Equal(t, info.PID, info2.PID)

	// stop twice
	info3, err := launcherCl.StopPlugin("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.Pid, info3.Pid)
	// stopping is idempotent
	info4, err := launcherCl.StopPlugin("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.Pid, info4.Pid)
}

func TestStartStopAll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	co1, cancelFunc := startService()
	defer cancelFunc()
	assert.NotNil(t, co1)

	launcherCl := launcher_client.NewLauncherClient("", co1)
	_, err := launcherCl.StartPlugin("yes")
	assert.NoError(t, err)

	// result should be 1 service running
	pluginList, err := launcherCl.ListPlugins(true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pluginList))

	// stopping
	err = launcherCl.StopAllPlugins(false)
	assert.NoError(t, err)

	// result should be no service running
	pluginList, err = launcherCl.ListPlugins(true)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pluginList))
	assert.NoError(t, err)
}
