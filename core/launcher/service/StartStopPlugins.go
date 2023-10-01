package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/struCoder/pidusage"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/samber/lo"
)

// StartAllPlugins starts all enabled plugins
func (svc *LauncherService) StartAllPlugins() (err error) {
	slog.Info("StartAll. Starting core and all enabled plugins")

	// start services in order from config
	for _, svcName := range svc.cfg.Autostart {
		svcInfo := svc.plugins[svcName]
		if svcInfo != nil && svcInfo.Running {
			// skip when already running
		} else {
			_, err2 := svc.StartPlugin(svcName)
			if err2 != nil {
				err = err2
			}
		}
	}
	// start the remaining plugins
	for svcName, svcInfo := range svc.plugins {
		if !svcInfo.Running {
			_, err2 := svc.StartPlugin(svcName)
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
func (svc *LauncherService) StartPlugin(pluginID string) (info launcher.PluginInfo, err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	// step 1: pre-checks
	serviceInfo, found := svc.plugins[pluginID]
	if !found {
		info.Status = fmt.Sprintf("plugin '%s' not found", pluginID)
		slog.Error(info.Status)
		return info, errors.New(info.Status)
	}
	if serviceInfo.Running {
		slog.Info("StartPlugin: Plugin is already running",
			"pluginID", pluginID, "StartTime", serviceInfo.StartTime)
		return *serviceInfo, nil
	}
	// don't start twice
	for _, cmd := range svc.cmds {
		if cmd.Path == serviceInfo.Path {
			err = fmt.Errorf("process for service '%s' already exists using PID %d",
				serviceInfo.Name, cmd.Process.Pid)
			slog.Error(err.Error())
			return *serviceInfo, err
		}
	}

	// step 2: create the command to start the service ... but wait for step 5
	svcCmd := exec.Command(serviceInfo.Path)

	// step3: setup logging before starting service
	if svc.cfg.LogPlugins {
		// inspired by https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
		logfile := path.Join(svc.f.Logs, pluginID+".log")
		fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			if svc.cfg.AttachStderr {
				// log stderr to launcher stderr and to file
				multiwriter := io.MultiWriter(os.Stderr, fp)
				svcCmd.Stderr = multiwriter
			} else {
				// just log stderr to file
				svcCmd.Stderr = fp
			}
			if svc.cfg.AttachStdout {
				// log stdout to launcher stdout and to file
				multiwriter := io.MultiWriter(os.Stdout, fp)
				svcCmd.Stdout = multiwriter
			} else {
				// just log stdout to file
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
	// step 4: generate the plugin credentials if needed
	if pluginID != CoreID {
		keyPath := path.Join(svc.f.Certs, pluginID+".key")
		tokenPath := path.Join(svc.f.Certs, pluginID+".token")

		slog.Info("Adding plugin user with key and token",
			"pluginID", pluginID, "keyPath", keyPath, "tokenPath", tokenPath)

		_, pubKey, err := svc.hc.LoadCreateKey(keyPath)
		if err != nil {
			slog.Error("Fail saving key for client. Continuing... ",
				"err", err, "pluginID", pluginID)
		}
		token, err := svc.authSvc.AddUser(
			pluginID, "plugin", "", pubKey, auth.ClientRoleService)
		if err != nil {
			slog.Error("Unable to add plugin to hub and create credentials. Continuing anyways", "err", err)
		} else {
			// save the auth token
			err = os.WriteFile(tokenPath, []byte(token), 0400)
		}
	}

	// step 5: start the command and setup serviceInfo
	slog.Info("Starting plugin", "pluginID", pluginID)
	err = svcCmd.Start()
	if err != nil {
		serviceInfo.Status = fmt.Sprintf("failed starting '%s': %s", pluginID, err.Error())
		err = errors.New(serviceInfo.Status)
		slog.Error(err.Error())
		return *serviceInfo, err
	}
	svc.cmds = append(svc.cmds, svcCmd)
	//slog.Warning("Service has started", "serviceName",pluginID)

	serviceInfo.StartTime = time.Now().Format(time.RFC3339)
	serviceInfo.PID = svcCmd.Process.Pid
	serviceInfo.Status = ""
	serviceInfo.StartCount++
	serviceInfo.Running = true

	// step 6: handle command termination and cleanup
	go func() {
		// cleanup after the process ends
		status := svcCmd.Wait()
		_ = status
		svc.mux.Lock()
		defer svc.mux.Unlock()

		serviceInfo.StopTime = time.Now().Format(time.RFC3339)
		serviceInfo.Running = false
		// processState holds exit info
		procState := svcCmd.ProcessState

		if status != nil {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped with: %s", pluginID, status.Error())
		} else if procState != nil {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped with exit code %d: sys='%v'", pluginID, procState.ExitCode(), procState.Sys())
		} else {
			serviceInfo.Status = fmt.Sprintf("Service '%s' has stopped without info", pluginID)
		}
		slog.Warn(serviceInfo.Status)
		svc.updateStatus(serviceInfo)
		// find the service to delete
		i := lo.IndexOf(svc.cmds, svcCmd)
		//lo.Delete(svc.cmds, i)  - why doesn't this exist?
		svc.cmds = append(svc.cmds[:i], svc.cmds[i+1:]...) // this is so daft!
	}()

	// Give it some time to get up and running in case it is needed as a dependency
	// TODO: wait for channel
	time.Sleep(time.Millisecond * 100)

	// last, update the CPU and memory status
	svc.updateStatus(serviceInfo)
	slog.Info("Plugin startup complete", "pluginID", pluginID)
	return *serviceInfo, err
}

// StopAllPlugins stops all running plugins in reverse order they were started
func (svc *LauncherService) StopAllPlugins() (err error) {

	svc.mux.Lock()
	slog.Info("Stopping all plugins", "count", len(svc.cmds))

	// use a copy of the commands as the command list will be mutated
	cmdsToStop := svc.cmds[:]

	svc.mux.Unlock()

	// stop each service
	for i := len(cmdsToStop) - 1; i >= 0; i-- {
		c := cmdsToStop[i]
		err = Stop(c.Path, c.Process.Pid)
	}
	time.Sleep(time.Millisecond)
	return err
}

func (svc *LauncherService) StopPlugin(name string) (info launcher.PluginInfo, err error) {
	slog.Info("Stopping service", "name", name)

	svc.mux.Lock()
	serviceInfo, found := svc.plugins[name]
	svc.mux.Unlock()
	if !found {
		info.Status = fmt.Sprintf("service '%s' not found", name)
		err = errors.New(info.Status)
		slog.Error("service not found", "name", name)
		return info, err
	}
	err = Stop(serviceInfo.Name, serviceInfo.PID)
	if err == nil {
		svc.mux.Lock()
		serviceInfo.Running = false
		serviceInfo.Status = "stopped by user"
		defer svc.mux.Unlock()
	}
	return *serviceInfo, err
}

// updateStatus updates the service  status
func (svc *LauncherService) updateStatus(svcInfo *launcher.PluginInfo) {
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
