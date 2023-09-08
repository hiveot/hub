package launcher_test

import (
	"github.com/hiveot/hub/core/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/core/launcher/service"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/logging"
)

var homeFolder = "/tmp"
var logFolder = "/tmp"

func newServer() (l launcher.ILauncher, stopFn func()) {
	var launcherConfig = config.NewLauncherConfig()
	launcherConfig.AttachStderr = true
	launcherConfig.AttachStdout = false
	launcherConfig.LogServices = true
	var f = utils.GetFolders(homeFolder, false)
	f.Plugins = "/bin" // for /bin/yes
	f.Logs = logFolder

	//ctx, cancelFunc := context.WithCancel(context.Background())
	svc := service.NewLauncherService(f, launcherConfig)
	err := svc.Start()
	if err != nil {
		slog.Error(err.Error())
	}

	return svc, func() {
		_ = svc.StopAll()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	svc, cancelFunc := newServer()
	defer cancelFunc()
	assert.NotNil(t, svc)
}

func TestList(t *testing.T) {
	svc, cancelFunc := newServer()
	defer cancelFunc()
	require.NotNil(t, svc)
	info, err := svc.List(false)
	assert.NoError(t, err)
	assert.NotNil(t, info)
}

func TestStartYes(t *testing.T) {
	// remove logfile from previous run
	logFile := path.Join(logFolder, "yes.log")
	_ = os.Remove(logFile)

	//
	svc, cancelFunc := newServer()
	defer cancelFunc()

	assert.NotNil(t, svc)
	info, err := svc.StartService("yes")
	require.NoError(t, err)
	assert.True(t, info.Running)
	assert.True(t, info.PID > 0)
	assert.True(t, info.StartTime != "")
	assert.FileExists(t, logFile)

	info2, err := svc.StopService("yes")
	time.Sleep(time.Millisecond * 10)
	assert.NoError(t, err)
	assert.False(t, info2.Running)
	assert.True(t, info2.StopTime != "")
}

func TestStartBadName(t *testing.T) {
	svc, cancelFunc := newServer()
	defer cancelFunc()
	assert.NotNil(t, svc)

	_, err := svc.StartService("notaservicename")
	require.Error(t, err)
	//
	_, err = svc.StopService("notaservicename")
	require.Error(t, err)
}

func TestStartStopTwice(t *testing.T) {
	svc, cancelFunc := newServer()
	defer cancelFunc()
	assert.NotNil(t, svc)

	info, err := svc.StartService("yes")
	assert.NoError(t, err)
	// again
	info2, err := svc.StartService("yes")
	assert.Error(t, err)
	_ = info2
	//assert.Equal(t, info.PID, info2.PID)

	// stop twice
	info3, err := svc.StopService("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.PID, info3.PID)
	// stopping is idempotent
	info4, err := svc.StopService("yes")
	assert.NoError(t, err)
	assert.False(t, info3.Running)
	assert.Equal(t, info.PID, info4.PID)
}

func TestStartStopAll(t *testing.T) {
	svc, cancelFunc := newServer()
	defer cancelFunc()
	assert.NotNil(t, svc)

	_, err := svc.StartService("yes")
	assert.NoError(t, err)

	// result should be 1 service running
	info, err := svc.List(true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(info))

	// stopping
	err = svc.StopAll()
	assert.NoError(t, err)

	// result should be no service running
	info, err = svc.List(true)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(info))

}
