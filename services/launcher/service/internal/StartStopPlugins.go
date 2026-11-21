package internal

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

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
	"github.com/struCoder/pidusage"

	"github.com/samber/lo"
)

// Update the account for a plugin
// 1. Create a key-pair if it doesn't exist and save it in the certs directory.
// 2. If the plugin account does not exist, add the account.
// 3. If the plugin account exists but a key is created, update the account public key.
func (svc *LauncherServiceImpl) _updatePluginAccount(certsDir string, pluginID string) error {

	var newKey bool

	// 1. Create a key-pair if it doesn't exist and save it in the certs directory.
	keyFile := path.Join(certsDir, pluginID+api.DefaultKeyFileSuffix)

	certKey, err := utils.LoadPrivateKey(keyFile)
	if err != nil {
		certKey = utils.NewEd25519Key()
		utils.SavePrivateKey(certKey, keyFile)
		newKey = true
	}

	// 2. If the plugin account does not exist, add the account.
	acctProfile, err := svc.authnAdmin.GetProfile(pluginID)
	if err != nil {
		slog.Info("Adding plugin account", "pluginID", pluginID)
		_, err = svc.authnAdmin.AddClient(pluginID, pluginID, api.ClientRoleService)
	} else if newKey {
		acctProfile.PubKeyPem = utils.PublicKeyToPem(certKey.Public())
		err = svc.authnAdmin.UpdateClientProfile(acctProfile)
	}

	// 3. Refresh the client certificate
	certFile := path.Join(certsDir, pluginID+api.DefaultCertFileSuffix)
	slog.Info("Refreshing plugin client cert", "pluginID", pluginID, "certsDir", certsDir)
	pluginCert, err := svc.certsClient.CreateClientCert(pluginID, api.ClientRoleService, 0, certKey.Public())
	if err == nil {
		err = utils.SaveX509Cert(pluginCert, certFile)
	}
	return err
}

// _startPlugin starts the plugin with the given name
// This creates a plugin authentication key and token files in the credentials directory (certs)
// before starting the plugin
//
// # This passes the server URL to the plugin using the --server
//
// This places a mux lock until start is complete.
func (svc *LauncherServiceImpl) _startPlugin(pluginID string) (pi launcher.PluginInfo, err error) {

	cfg := svc.cfg
	env := svc.env

	// step 1: pre-launch checks
	svc.mux.Lock()
	pluginInfo, found := svc.plugins[pluginID]
	svc.mux.Unlock()
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
	svcCmd := exec.Command(pluginInfo.ExecPath, "localhost")
	svcCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	// forward the serverURL argument if provided
	if env.ServerURL != "" && cfg.ProvideServerURL {
		svcCmd.Args = append(svcCmd.Args, fmt.Sprintf("--serverURL=%s", env.ServerURL))
	}
	// forward a Thing Directory TD url if provided
	if env.DirTDURL != "" && cfg.ProvideDirectoryURL {
		svcCmd.Args = append(svcCmd.Args, fmt.Sprintf("--dirTD=%s", env.DirTDURL))
	}

	// step3: setup logging before starting service
	if cfg.LogPlugins {
		// set default plugin loglevel using environment variable LOGLEVEL. See GetAppEnvironment
		svcCmd.Env = append(os.Environ(), "LOGLEVEL="+cfg.LogLevel)

		// inspired by https://gist.github.com/jerblack/4b98ba48ed3fb1d9f7544d2b1a1be287
		logfile := path.Join(env.LogsDir, pluginID+".log")
		_ = os.MkdirAll(env.LogsDir, 0700)
		fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			if cfg.AttachStderr {
				// log stderr to launcher stderr and to file
				multiwriter := io.MultiWriter(os.Stderr, fp)
				svcCmd.Stderr = multiwriter
				slog.Info("attaching stderr using multiwriter")
			} else {
				// just log stderr to file
				slog.Info("attaching stderr using logfile", "logfile", logfile)
				svcCmd.Stderr = fp
			}
			if cfg.AttachStdout {
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
		if cfg.AttachStderr {
			svcCmd.Stderr = os.Stderr
		}
		if cfg.AttachStdout {
			svcCmd.Stdout = os.Stdout
		}
	}
	// step 4: add the plugin account and generate its credentials
	//  (this does not apply to the gateway runtime itself as it provides authn access)
	if pluginID != cfg.RuntimeBin && cfg.CreatePluginCred {
		err = svc._updatePluginAccount(env.CertsDir, pluginID)
		if err != nil {
			slog.Error(err.Error())
			return pi, err
		}
	}

	// step 5: start the plugin
	err = svcCmd.Start()
	if err != nil {
		pluginInfo.Status = fmt.Sprintf("failed starting '%s': %s", pluginID, err.Error())
		err = errors.New(pluginInfo.Status)
		slog.Error(err.Error())
		svc.mux.Lock()
		svc.plugins[pluginID] = pluginInfo
		svc.mux.Unlock()

		return pluginInfo, err
	}
	svc.cmds = append(svc.cmds, svcCmd)
	//slog.Warning("Service has started", "serviceName",pluginID)

	pluginInfo.StartedTime = utils.FormatNowUTCMilli()
	pluginInfo.Pid = int64(svcCmd.Process.Pid)
	pluginInfo.Status = ""
	pluginInfo.StartCount++
	pluginInfo.Running = true

	// step 6: handle command termination and cleanup in the background
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

	// Give it some minimal time to get up and running in case this service is needed as a
	// dependency of another or fails on start.
	time.Sleep(time.Millisecond * time.Duration(time.Duration(cfg.StartWait)))

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
	svc.mux.Lock()
	svc.plugins[pluginID] = pluginInfo
	svc.mux.Unlock()
	return pluginInfo, err
}

// Start the runtime, if configured
// This returns nil if no runtime is configured.
func (svc *LauncherServiceImpl) _startRuntime() (err error) {
	runtimeBin := svc.cfg.RuntimeBin
	if runtimeBin == "" {
		return nil
	}

	svc.mux.Lock()
	_, foundRuntime := svc.plugins[runtimeBin]
	svc.mux.Unlock()
	if !foundRuntime {
		return nil
	}
	if foundRuntime {
		// runtime is added and starts first
		pi, err := svc._startPlugin(runtimeBin)
		if err != nil {
			slog.Error("Starting runtime failed", "runtimeBin", runtimeBin, "err", err)
			return err
		} else {
			slog.Warn("Runtime started successfully", "runtimeBin", runtimeBin)

			// a bit of a stopgap for slow systems
			// FIXME: runtime should signal readiness
			time.Sleep(time.Millisecond * time.Duration(svc.cfg.RuntimeWait))

			// some extra time if its still busy
			if pi.Cpu >= 99 {
				time.Sleep(time.Second)
			}
		}
		// since the runtime is launched locally a local connection is the most efficient
		// unless a server address is already configured.
		if svc.env.ServerURL == "" {
			svc.env.ServerURL = DefaultLocalServerURL
		}
	}
	return nil
}

// StartAllPlugins starts all enabled plugins
func (svc *LauncherServiceImpl) StartAllPlugins() (err error) {

	slog.Info("StartAllPlugins")
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
func (svc *LauncherServiceImpl) StartPlugin(pluginID string) (launcher.PluginInfo, error) {

	slog.Info("StartPlugin",
		slog.String("pluginID", pluginID),
	)

	pluginInfo, err := svc._startPlugin(pluginID)
	if err != nil {
		// notify subscribers publish a started event
		go func() {
			svc.PubEvent(launcher.PluginStartedEvent, pluginID, true)
		}()
	}
	return pluginInfo, err
}

// StopAllPlugins stops all running plugins in reverse order they were started
// If fullStop is set then also stop the runtime.
func (svc *LauncherServiceImpl) StopAllPlugins(fullStop bool) (err error) {

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
		if !fullStop && svc.cfg.RuntimeBin != "" && strings.HasSuffix(c.Path, svc.cfg.RuntimeBin) {
			// don't stop the runtime as that would render things unreachable
			slog.Info("Not stopping the core", "path", c.Path)
		} else {
			err = StopProcess(c.Path, c.Process.Pid)
			// TODO: publish a stopped event
		}
	}
	time.Sleep(time.Millisecond)
	return err
}

func (svc *LauncherServiceImpl) StopPlugin(pluginID string) (pluginInfo launcher.PluginInfo, err error) {
	slog.Info("Stopping plugin",
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
	err = StopProcess(pluginID, int(pluginInfo.Pid))

	svc.mux.Lock()
	defer svc.mux.Unlock()
	// reload the plugin status
	pluginInfo, _ = svc.plugins[pluginID]
	pluginInfo.Status = "Stopped"
	svc.plugins[pluginID] = pluginInfo

	// notify subscribers publish a stopped event
	go func() {
		svc.PubEvent(launcher.PluginStoppedEvent, pluginID, true)
	}()

	return pluginInfo, nil
}

// updateStatus updates the service running cpu/mem in the  status
func (svc *LauncherServiceImpl) updateStatus(svcInfo *launcher.PluginInfo) {
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
