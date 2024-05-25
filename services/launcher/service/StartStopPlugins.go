package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/services/launcher/launcherapi"
	"github.com/struCoder/pidusage"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/samber/lo"
)

// _startPlugin starts the plugin with the given name
// This creates a plugin authentication key and token files in the credentials directory (certs)
// before starting the plugin
// This places a mux lock until start is complete.
func (svc *LauncherService) _startPlugin(pluginName string) (pi launcherapi.PluginInfo, err error) {

	svc.mux.Lock()
	defer svc.mux.Unlock()

	// step 1: pre-checks
	pluginInfo := svc.plugins[pluginName]
	if pluginInfo == nil {
		err = fmt.Errorf("plugin ID '%s' not found", pluginName)
		slog.Error("_startPlugin: plugin not found", "name", pluginName)
		return pi, err
	}
	if pluginInfo.Running {
		slog.Info("_startPlugin: Plugin is already running",
			slog.String("pluginName", pluginName),
			slog.String("StartTime", utils.FormatMSE(pluginInfo.StartTimeMSE, true)))
		return *pluginInfo, nil
	}
	slog.Warn("_startPlugin", "pluginName", pluginName, "path", pluginInfo.Path)

	// don't start twice
	for _, cmd := range svc.cmds {
		if cmd.Path == pluginInfo.Path {
			err := fmt.Errorf("process for service '%s' already exists using PID %d",
				pluginInfo.Name, cmd.Process.Pid)
			slog.Error(err.Error())
			return *pluginInfo, err
		}
	}

	// step 2: create the command to start the service ... but wait for step 5
	svcCmd := exec.Command(pluginInfo.Path)
	svcCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	// step3: setup logging before starting service
	if svc.cfg.LogPlugins {
		// set default plugin loglevel using environment variable LOGLEVEL. See GetAppEnvironment
		svcCmd.Env = append(os.Environ(), "LOGLEVEL="+svc.cfg.LogLevel)

		// inspired by https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
		logfile := path.Join(svc.env.LogsDir, pluginName+".log")
		_ = os.MkdirAll(svc.env.LogsDir, 0700)
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
	// step 4: add the serviceID as a client and generate its credentials
	if pluginName != svc.cfg.RuntimeBin {
		tokenPath := path.Join(svc.env.CertsDir, pluginName+".token")

		slog.Info("Adding plugin service client with key and token",
			"pluginName", pluginName, "certsDir", svc.env.CertsDir, "tokenPath", tokenPath)

		// add a service account and generate a new token file in the keys directory
		// the service must have read access to this directory, or the keys must be
		// copied elsewhere by the administrator.
		_, err = svc.mngAuth.AddAgent(api.ClientTypeService, pluginName, pluginName, "")
		if err != nil {
			slog.Error("Unable to add plugin to hub and create credentials. Continuing anyways", "err", err)
		}
	}

	// step 5: start the command and setup pluginInfo
	err = svcCmd.Start()
	if err != nil {
		pluginInfo.Status = fmt.Sprintf("failed starting '%s': %s", pluginName, err.Error())
		err = errors.New(pluginInfo.Status)
		slog.Error(err.Error())
		return *pluginInfo, err
	}
	svc.cmds = append(svc.cmds, svcCmd)
	//slog.Warning("Service has started", "serviceName",pluginID)

	pluginInfo.StartTimeMSE = time.Now().UnixMilli()
	pluginInfo.PID = svcCmd.Process.Pid
	pluginInfo.Status = ""
	pluginInfo.StartCount++
	pluginInfo.Running = true

	// step 6: handle command termination and cleanup
	var startStatus error
	go func() {
		// cleanup after the process ends
		startStatus = svcCmd.Wait()
		svc.mux.Lock()
		defer svc.mux.Unlock()
		pluginInfo.StopTimeMSE = time.Now().UnixMilli()
		pluginInfo.Running = false
		// processState holds exit info
		procState := svcCmd.ProcessState

		if startStatus != nil {
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped with: %s",
				pluginName, startStatus.Error())
		} else if procState != nil {
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped with exit code %d: sys='%v'",
				pluginName, procState.ExitCode(), procState.Sys())
		} else {
			pluginInfo.Status = fmt.Sprintf("Plugin '%s' has stopped without info", pluginName)
		}
		slog.Warn("Plugin has stopped",
			slog.String("pluginName", pluginName),
			slog.String("status", pluginInfo.Status))
		svc.updateStatus(pluginInfo)
		// find the service to delete
		i := lo.IndexOf(svc.cmds, svcCmd)
		//lo.Delete(svc.cmds, i)  - why doesn't this exist?
		svc.cmds = append(svc.cmds[:i], svc.cmds[i+1:]...) // this is so daft!
	}()

	// Give it some time to get up and running in case it is needed as a dependency
	// TODO: wait for channel
	time.Sleep(time.Millisecond * 100)

	// check if its still running
	if startStatus != nil {
		// something went wrong
		err = startStatus
	}

	// last, update the CPU and memory status
	svc.updateStatus(pluginInfo)
	if err != nil {
		slog.Error("Plugin startup failed", "pluginName", pluginName, "err", err, "status", pluginInfo.Status)
	}
	return *pluginInfo, err
}

// StartAllPlugins starts all enabled plugins
func (svc *LauncherService) StartAllPlugins() (err error) {
	slog.Info("StartAll. Starting core and all enabled plugins")

	// start services in order from config
	for _, pluginName := range svc.cfg.Autostart {
		svcInfo := svc.plugins[pluginName]
		if svcInfo != nil && svcInfo.Running {
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
func (svc *LauncherService) StartPlugin(args launcherapi.StartPluginArgs) (launcherapi.StartPluginResp, error) {

	slog.Warn("StartPlugin",
		slog.String("pluginID", args.Name),
	)

	pluginInfo, err := svc._startPlugin(args.Name)
	resp := launcherapi.StartPluginResp{PluginInfo: pluginInfo}
	return resp, err
}

// StopAllPlugins stops all running plugins in reverse order they were started
// If includingCore is set then also stop the core.
func (svc *LauncherService) StopAllPlugins(args *launcherapi.StopAllPluginsArgs) (err error) {

	svc.mux.Lock()

	// use a copy of the commands as the command list will be mutated
	cmdsToStop := svc.cmds[:]
	slog.Info("Stopping all plugins",
		slog.Int("count", len(cmdsToStop)),
	)

	svc.mux.Unlock()

	// stop each service in reverse order
	for i := len(cmdsToStop) - 1; i >= 0; i-- {
		c := cmdsToStop[i]
		if !args.IncludingRuntime && svc.cfg.RuntimeBin != "" && strings.HasSuffix(c.Path, svc.cfg.RuntimeBin) {
			// don't stop the core as that would render things unreachable
			slog.Info("Not stopping the core", "path", c.Path)
		} else {
			err = Stop(c.Path, c.Process.Pid)
		}
	}
	time.Sleep(time.Millisecond)
	return err
}

func (svc *LauncherService) StopPlugin(args launcherapi.StopPluginArgs) (resp launcherapi.StopPluginResp, err error) {

	svc.mux.Lock()
	pluginInfo, _ := svc.plugins[args.Name]
	svc.mux.Unlock()
	if pluginInfo == nil {
		err = fmt.Errorf("plugin '%s' not found", args.Name)
		slog.Error("Plugin not found", "pluginName", args.Name)
		return resp, err
	}
	err = Stop(pluginInfo.Name, pluginInfo.PID)

	svc.mux.Lock()
	defer svc.mux.Unlock()
	pluginInfo.Running = false
	// stoptime is set when process stops
	if err != nil {
		pluginInfo.Status = err.Error()
	} else {
		pluginInfo.Status = "stopped by user"
	}

	resp.PluginInfo = *pluginInfo
	return resp, err
}

// updateStatus updates the service  status
func (svc *LauncherService) updateStatus(svcInfo *launcherapi.PluginInfo) {
	if svcInfo.PID != 0 {

		//Option A: use pidusage - doesn't work on Windows though
		//warning, pidusage is not very fast
		pidStats, _ := pidusage.GetStat(svcInfo.PID)
		if pidStats != nil {
			svcInfo.RSS = int(pidStats.Memory) // RSS is in KB
			svcInfo.CPU = int(pidStats.CPU)
		} else {
			svcInfo.CPU = 0
			svcInfo.RSS = 0
		}

		// Option B: use go-osstat - slower
		//cpuStat, err := cpu.Get()
		//if err == nil {
		//	svcInfo.CPU = cpuStat.CPUCount // FIXME: this is a counter, not %
		//}
		//memStat, err := memory.Get()
		//if err == nil {
		//	svcInfo.RSS = int(memStat.Used)
		//}

		//Option C: read statm directly. Fastest but only gets memory.
		//path := fmt.Sprintf("/proc/%d/statm", svcInfo.PID)
		//statm, err := ioutil.ReadFile(path)
		//if err == nil {
		//	fields := strings.Split(string(statm), " ")
		//	if len(fields) < 2 {
		//		// invalid data
		//	} else {
		//		rss, err := strconv.ParseInt(fields[1], 10, 64)
		//		if err != nil {
		//			// invalid data
		//		} else {
		//			svcInfo.RSS = int(rss * int64(os.Getpagesize()))
		//		}
		//	}
		//}
	}

}
