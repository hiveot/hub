package launcher_test

import (
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
	launcher_client "github.com/hiveot/hub/services/launcher/client"
	launcher_service "github.com/hiveot/hub/services/launcher/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var homeDir = "/tmp/hiveot/launcher-test"
var env *api.AppEnvironment

// var logDir = "/tmp/hiveot/launcher-test"

// the following are set by the testmain
// var testEnv *testenv.TestEnv

// const agentUsesWSS = false

//var testClients = []msgserver.ClientAuthInfo{{
//	SenderID:   launcher.ServiceName,
//	ClientType: auth.ClientTypeService,
//	Role:       auth.ClientRoleService,
//}, {
//	SenderID:   testenv.TestAdminUserID,
//	ClientType: auth.ClientTypeUser,
//	Role:       auth.ClientRoleAdmin,
//}}

func startService() (cl *launcher_client.LauncherClient, stopFn func()) {
	const launcherID = launcher.LauncherServiceCellType
	const adminID = "admin"

	env = api.NewAppEnvironment(homeDir, false)

	os.MkdirAll(env.HomeDir, 0750)
	os.MkdirAll(env.PluginsDir, 0750)
	os.MkdirAll(env.LogsDir, 0750)
	// use the 'ping' package
	pingApp, err := os.ReadFile("/usr/bin/ping")
	err = os.WriteFile(path.Join(env.PluginsDir, "ping"), pingApp, 0755)

	// testEnv = testenv.NewTestEnv(true)
	// srv := testEnv.StartTestServer("")

	// authn service is needed to create auth tokens
	// tokenDir := testEnv.Env.CertsDir
	authnStoreDir := env.GetStorageDir(authn.AuthnServiceCellType)
	authnSvc, err := authn_service.NewAuthnService(env.CertsDir, authnStoreDir, false)

	// certs service is need to create service certificates
	certsCfg := &certs.CertsConfig{}
	certsSvc, err := certs_service.NewCertsService(env, certsCfg)

	// directory is needed to write TD (optional - WriteTD)
	// dirSvc, err := directory_service.NewDirectoryService("", "", nil)
	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}

	// create the launcher service itself
	var launcherConfig = launcher.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogPlugins = true
	launcherConfig.ProvideDirectoryURL = false
	launcherConfig.ProvideServerURL = false
	launcherConfig.WriteTD = false

	svc, err := launcher_service.NewLauncherService(env, launcherConfig)
	if err != nil {
		slog.Error(err.Error())
		panic(err.Error())
	}

	// link the launcher to -> authn -> certs
	svc.SetRequestSink(authnSvc)
	authnSvc.SetRequestSink(certsSvc)

	svc.Start()
	// the agent adds a sendersID
	agent := testenv.NewTestTransport(launcherID, svc)
	//--- connect the launcher client
	cl = launcher_client.NewLauncherClient("", agent)
	return cl, func() {
		cl.Stop()
		svc.Stop()
		authnSvc.Stop()
		certsSvc.Stop()
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

	launcherCl, cancelFunc := startService()
	defer cancelFunc()
	// expect the 'ping' plugin copied in startService
	infoList, err := launcherCl.ListPlugins(false)
	require.NoError(t, err)
	assert.Equal(t, 1, len(infoList))
}

// func TestListNoPermission(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())
// 	userID := "user1"

// 	co1, cancelFunc := startService()
// 	defer cancelFunc()
// 	require.NotNil(t, co1)

// 	cc2, _ := testEnv.NewTestClient(userID, api.ClientRoleNone)
// 	cc2.Connect()
// 	defer cc2.Close()
// 	launcherCl := launcher_client.NewLauncherClient("", cc2)
// 	info2, err := launcherCl.ListPlugins(false)
// 	// FIXME: test set authz
// 	require.Error(t, err, "user without role should not be able to use launcher")
// 	require.Empty(t, info2)
// }

func TestStartPing(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const appName = "ping"

	//
	launcherCl, cancelFunc := startService()
	defer cancelFunc()

	// remove logfile from previous run
	logFile := path.Join(env.LogsDir, appName+".log")
	_ = os.Remove(logFile)

	info, err := launcherCl.StartPlugin(appName)
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.Pid > 0)
	assert.True(t, info.StartedTime != "")
	assert.FileExists(t, logFile)

	time.Sleep(time.Millisecond * 1)

	info2, err := launcherCl.StopPlugin(appName)
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

	info, err := launcherCl.StartPlugin("ping")
	assert.NoError(t, err)
	// second start will just return
	info2, err := launcherCl.StartPlugin("ping")
	assert.NoError(t, err)
	_ = info2
	//assert.Equal(t, info.PID, info2.PID)

	// stop twice
	info3, err := launcherCl.StopPlugin("ping")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.Pid, info3.Pid)
	// stopping is idempotent
	info4, err := launcherCl.StopPlugin("ping")
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
	_, err := launcherCl.StartPlugin("ping")
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
