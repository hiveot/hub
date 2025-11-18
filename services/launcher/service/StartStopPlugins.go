package service

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/hiveot/hivehub/lib/plugin"
	authn "github.com/hiveot/hivehub/runtime/authn/api"
	launcher "github.com/hiveot/hivehub/services/launcher/api"
	"github.com/hiveot/hivekitgo/utils"
	"github.com/struCoder/pidusage"

	"github.com/samber/lo"
)

// _startPlugin starts the plugin with the given name
// This creates a plugin authentication key and token files in the credentials directory (certs)
// before starting the plugin
//
// # This passes the server URL to the plugin using the --server
//
// This places a mux lock until start is complete.
func (svc *LauncherService) _startPlugin(pluginID string) (pi launcher.PluginInfo, err error) {

	svc.mux.Lock()
	defer svc.mux.Unlock()

	// step 1: pre-launch checks
	pluginInfo, found := svc.plugins[pluginID]
	if !found {
		err = fmt.Errorf("plugin ID '%s' not found", pluginID)
		slog.Error("_startPlugin: plugin not found", "name", pluginID)
		return pi, err
	}
	if pluginInfo.Running {
		slog.Info("_startPlugin: Plugin is already running",
			slog.String("pluginID", pluginID),
			slog.String("StartTime", pluginInfo.StartedTime))
		return pluginInfo, nil
	}
	//
	slog.Warn("_startPlugin", "pluginID", pluginID, "path", pluginInfo.ExecPath)

	// don't start twice
	for _, cmd := range svc.cmds {
		if cmd.Path == pluginInfo.ExecPath {
			err := fmt.Errorf("process for service '%s' already exists using PID %d",
				pluginID, cmd.Process.Pid)
			slog.Error(err.Error())
			return pluginInfo, err
		}
	}

	// step 2: create the command to start the service ... but wait for step 5
	svcCmd := exec.Command(pluginInfo.ExecPath)
	svcCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	// provide a server url (deprecated - migrating to directory)
	if svc.serverURL != "" && svc.cfg.ProvideServerURL {
		svcCmd.Args = append(svcCmd.Args, fmt.Sprintf("--serverURL=%s", svc.serverURL))
	}
	// provide a directory url
	if svc.directoryURL != "" && svc.cfg.ProvideDirectoryURL {
		svcCmd.Args = append(svcCmd.Args, fmt.Sprintf("--%s=%s", plugin.DirectoryURL_Arg, svc.directoryURL))
	}

	// step3: setup logging before starting service
	if svc.cfg.LogPlugins {
		// set default plugin loglevel using environment variable LOGLEVEL. See GetAppEnvironment
		svcCmd.Env = append(os.Environ(), "LOGLEVEL="+svc.cfg.LogLevel)

		// inspired by https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
		logfile := path.Join(svc.cfg.LogsDir, pluginID+".log")
		_ = os.MkdirAll(svc.cfg.LogsDir, 0700)
		fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			if svc.cfg.AttachStderr {
				// log stderr to launcher stderr and to file
				multiwriter := io.MultiWriter(os.Stderr, fp)
				svcCmd.Stderr = multiwriter
				slog.Info("attaching stderr using multiwriter")
			} else {
				// just log stderr to file
				slog.Info("attaching stderr using logfile", "logfile", logfile)
				svcCmd.Stderr = fp
			}
			if svc.cfg.AttachStdout {
				// log stdout to launcher stdout and to file
				multiwriter := io.MultiWriter(os.Stdout, fp)
				slog.Info("attaching stdout using multiwriter")
				svcCmd.Stdout = multiwriter
			} else {
				// just log stdout to file
				slog.Info("attaching stdout using logfile", "logfile", logfile)
				svcCmd.Stdout = fp
			}
		} else {
			slog.Error("creating logfile failed", "err", err, "file", logfile)
		}
	} else {
		if svc.cfg.AttachStderr {
			svcCmd.Stderr = os.Stderr
		}
		if svc.cfg.AttachStdout {
			svcCmd.Stdout = os.Stdout
		}
	}
	// step 4: add the service account and generate its credentials
	//  (this does not apply to the runtime itself)
	if pluginID != svc.cfg.RuntimeBin && svc.cfg.CreatePluginCred {
		tokenPath := path.Join(svc.certsDir, pluginID+".token")

		slog.Info("Adding plugin service client with key and token",
			"pluginID", pluginID, "certsDir", svc.certsDir, "tokenPath", tokenPath)

		// add a service account. Authn admin generates a new token file in the keys directory
		// the service must have read access to this directory, or the keys must be
		// copied elsewhere by the administrator.
		_, err = authn.AdminAddService(svc.ag.Consumer, pluginID, pluginID, "")
		if err != nil {
			slog.Error("Unable to add plugin to hub and create credentials. Continuing anyways", "err", err)
		}
	}

	// step 5: start the plugin
	err = svcCmd.Start()
	if err != nil {
		pluginInfo.Status = fmt.Sprintf("failed starting '%s': %s", pluginID, err.Error())
		err = errors.New(pluginInfo.Status)
		slog.Error(err.Error())
		svc.plugins[pluginID] = pluginInfo

		return pluginInfo, err
	}
	svc.cmds = append(svc.cmds, svcCmd)
	//slog.Warning("Service has started", "serviceName",pluginID)

	pluginInfo.StartedTime = utils.FormatNowUTCMilli()
	pluginInfo.Pid = int64(svcCmd.Process.Pid)
	pluginInfo.Status = ""
	pluginInfo.StartCount++
	pluginInfo.Running = true

	// step 6: handle command termination and cleanup
	var exitError error
	go func() {
		// cleanup after the process ends
		exitError = svcCmd.Wait()
		svc.mux.Lock()
		defer svc.mux.Unlock()
		pluginInfo.StoppedTime = utils.FormatNowUTCMilli()
		pluginInfo.Running = false
		// processState holds exit info
		procState := svcCmd.ProcessState

		if exitError != nil {
			// expect error is signal:terminated
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped with: %s",
				pluginID, exitError.Error())
		} else if procState != nil {
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped with exit code %d: sys='%v'",
				pluginID, procState.ExitCode(), procState.Sys())
		} else {
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped without info", pluginID)
		}
		slog.Warn("Plugin has stopped",
			slog.String("pluginID", pluginID),
			slog.String("status", pluginInfo.Status))
		svc.updateStatus(&pluginInfo)
		svc.plugins[pluginID] = pluginInfo

		// find the service to delete
		i := lo.IndexOf(svc.cmds, svcCmd)
		//lo.Delete(svc.cmds, i)  - why doesn't this exist?
		svc.cmds = append(svc.cmds[:i], svc.cmds[i+1:]...) // this is so daft!
	}()

	// Give it some minimal time to get up and running in case this service is needed as a dependency of another
	time.Sleep(time.Millisecond * time.Duration(svc.cfg.StartWait))

	// check if it is still running
	if exitError != nil {
		// something went wrong
		err = exitError
	}

	// TODO: publish an started event

	// last, update the CPU and memory status
	svc.updateStatus(&pluginInfo)
	if err != nil {
		slog.Error("Plugin startup failed", "pluginID", pluginID, "err", err, "status", pluginInfo.Status)
	} else {
		slog.Info("Plugin '"+pluginID+"' startup succeeded", "cpu", pluginInfo.Cpu)
	}
	svc.plugins[pluginID] = pluginInfo
	return pluginInfo, err
}

// StartAllPlugins starts all enabled plugins
func (svc *LauncherService) StartAllPlugins(senderID string) (err error) {

	slog.Info("StartAllPlugins",
		slog.String("senderID", senderID),
	)
	// start services in order from config
	for _, pluginName := range svc.cfg.Autostart {
		svcInfo, found := svc.plugins[pluginName]
		if found && svcInfo.Running {
			// skip when already running
		} else {
			_, err2 := svc._startPlugin(pluginName)

			if err2 != nil {
				err = err2
			}
		}
	}
	// start the remaining plugins
	for pluginName, svcInfo := range svc.plugins {
		if !svcInfo.Running {
			_, err2 := svc._startPlugin(pluginName)
			if err2 != nil {
				err = err2
			}
		}
	}
	return err
}

// StartPlugin starts the plugin with the given name
// This creates a plugin authentication key and token files in the credentials directory (certs)
// before starting the plugin.
func (svc *LauncherService) StartPlugin(senderID string, pluginID string) (launcher.PluginInfo, error) {

	slog.Info("StartPlugin",
		slog.String("senderID", senderID),
		slog.String("pluginID", pluginID),
	)

	pluginInfo, err := svc._startPlugin(pluginID)
	if err != nil {
		// notify subscribers publish a started event
		go func() {
			_ = svc.ag.PubEvent(launcher.AdminServiceID, launcher.AdminEventStarted, pluginID)
		}()
	}
	return pluginInfo, err
}

// StopAllPlugins stops all running plugins in reverse order they were started
// If includingCore is set then also stop the core.
func (svc *LauncherService) StopAllPlugins(senderID string, fullStop bool) (err error) {

	svc.mux.Lock()

	// use a copy of the commands as the command list will be mutated
	cmdsToStop := svc.cmds[:]
	slog.Info("Stopping all plugins",
		slog.String("senderID", senderID),
		slog.Int("count", len(cmdsToStop)),
	)

	svc.mux.Unlock()

	// stop each service in reverse order
	for i := len(cmdsToStop) - 1; i >= 0; i-- {
		c := cmdsToStop[i]
		if !fullStop && svc.cfg.RuntimeBin != "" && strings.HasSuffix(c.Path, svc.cfg.RuntimeBin) {
			// don't stop the runtime as that would render things unreachable
			slog.Info("Not stopping the core", "path", c.Path)
		} else {
			err = Stop(c.Path, c.Process.Pid)
			// TODO: publish a stopped event
		}
	}
	time.Sleep(time.Millisecond)
	return err
}

func (svc *LauncherService) StopPlugin(senderID string, pluginID string) (pluginInfo launcher.PluginInfo, err error) {
	slog.Info("Stopping plugin",
		slog.String("senderID", senderID),
		slog.String("pluginID", pluginID))

	svc.mux.Lock()
	pluginInfo, found := svc.plugins[pluginID]
	svc.mux.Unlock()
	if !found {
		err := fmt.Errorf("plugin '%s' not found", pluginID)
		slog.Error("Plugin not found", "pluginID", pluginID)
		return pluginInfo, err
	}
	// stop status is updated when the process ends
	err = Stop(pluginID, int(pluginInfo.Pid))

	svc.mux.Lock()
	defer svc.mux.Unlock()
	// reload the plugin status
	pluginInfo, _ = svc.plugins[pluginID]
	pluginInfo.Status = "Stopped by user: " + senderID
	svc.plugins[pluginID] = pluginInfo

	// notify subscribers publish a stopped event
	go func() {
		_ = svc.ag.PubEvent(launcher.AdminServiceID, launcher.AdminEventStopped, pluginID)
	}()

	return pluginInfo, nil
}

// updateStatus updates the service running cpu/mem in the  status
func (svc *LauncherService) updateStatus(svcInfo *launcher.PluginInfo) {
	if svcInfo.Pid != 0 {

		//Option A: use pidusage - doesn't work on Windows though
		//warning, pidusage is not very fast
		pid := int(svcInfo.Pid)
		pidStats, _ := pidusage.GetStat(pid)
		if pidStats != nil {
			svcInfo.Rss = int64(pidStats.Memory) // RSS is in KB
			svcInfo.Cpu = int64(pidStats.CPU)
		} else {
			svcInfo.Cpu = 0
			svcInfo.Rss = 0
		}

	}

}
