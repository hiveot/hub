package service

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/services/launcher/config"
	"github.com/hiveot/hub/services/launcher/launcherapi"
	"github.com/hiveot/hub/wot/protocolclients"
	"github.com/hiveot/hub/wot/protocolclients/connect"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// LauncherService manages starting and stopping of plugins
// This implements the ILauncher interface
type LauncherService struct {
	// service configuration
	cfg config.LauncherConfig
	env plugin.AppEnvironment

	// map of plugin name to running status
	plugins map[string]*launcherapi.PluginInfo
	// list of started commands in startup order
	cmds []*exec.Cmd

	// hub messaging client
	hc clients.IAgent

	// mutex to keep things safe
	mux sync.Mutex
	// watch plugin folders for updates
	serviceWatcher *fsnotify.Watcher
	// service is running
	isRunning atomic.Bool
	// closing channel
	done chan bool
}

// Add discovered runtime to svc.plugins
func (svc *LauncherService) addRuntime(runtimeBin string) error {
	if runtimeBin != "" {
		runtimePath := path.Join(svc.env.BinDir, runtimeBin)
		runtimeInfo, err := os.Stat(runtimePath)
		if err != nil {
			err = fmt.Errorf("addRuntime. runtime in config not found. Path=%s", runtimePath)
			return err
		}
		pluginInfo, found := svc.plugins[runtimeBin]
		if found {
			// update existing entry for runtime
			pluginInfo.ModifiedTime = runtimeInfo.ModTime().Format(time.RFC3339)
			pluginInfo.Size = runtimeInfo.Size()
		} else {
			// add new entry for runtime
			pluginInfo = &launcherapi.PluginInfo{
				Name:    runtimeInfo.Name(),
				Path:    runtimePath,
				Uptime:  0,
				Running: false,
			}
			pluginInfo.ModifiedTime = runtimeInfo.ModTime().Format(time.RFC3339)
			pluginInfo.Size = runtimeInfo.Size()
			svc.plugins[runtimeBin] = pluginInfo
		}
	}
	return nil
}

// Add newly discovered executable plugins to svc.plugins
// If the service is already know, only update its size and timestamp
func (svc *LauncherService) addPlugins(folder string) error {
	count := 0
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// ignore directories and non executable files
		fileInfo, err := entry.Info()
		if err != nil {
			slog.Error("Unable to read plugin info. Skipped", "err", err.Error())
		} else {
			size := fileInfo.Size()
			fileMode := fileInfo.Mode()
			isExecutable := fileMode&0100 != 0
			isFile := !entry.IsDir()
			if isFile && isExecutable && size > 0 {
				count++
				pluginInfo, found := svc.plugins[entry.Name()]
				if !found {
					pluginInfo = &launcherapi.PluginInfo{
						Name:    entry.Name(),
						Path:    path.Join(folder, entry.Name()),
						Uptime:  0,
						Running: false,
					}
					svc.plugins[pluginInfo.Name] = pluginInfo
				}
				pluginInfo.ModifiedTime = fileInfo.ModTime().Format(time.RFC3339)
				pluginInfo.Size = size
			}
		}
	}
	slog.Info("found plugins", "count", count, "directory", folder)
	return nil
}

// List all available or just the running plugins and their status
// This returns the list of plugins sorted by name
func (svc *LauncherService) List(args launcherapi.ListArgs) (launcherapi.ListResp, error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	// get the keys of the plugins to include and sort them
	keys := make([]string, 0, len(svc.plugins))
	for key, val := range svc.plugins {
		if !args.OnlyRunning || val.Running {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	infoList := make([]launcherapi.PluginInfo, 0, len(keys))
	for _, key := range keys {
		svcInfo := svc.plugins[key]
		svc.updateStatus(svcInfo)
		infoList = append(infoList, *svcInfo)
	}
	resp := launcherapi.ListResp{PluginInfoList: infoList}
	return resp, nil
}

// ScanPlugins scans the plugin folder for changes and updates the plugins list
func (svc *LauncherService) ScanPlugins() error {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	err := svc.addPlugins(svc.env.PluginsDir)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

// Start the launcher service
// This first starts the runtime defined in the config, then connects to the hub
// to be able to create auth keys and tokens, and to subscribe to rpc requests.
//
// Call stop to end
func (svc *LauncherService) Start() error {
	slog.Info("Starting LauncherService", "clientID", svc.env.ClientID)
	svc.isRunning.Store(true)

	// include the runtime
	runtimeBin := svc.cfg.RuntimeBin
	if runtimeBin != "" {
		err := svc.addRuntime(runtimeBin)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
	}
	// 1: determine the inventory of plugins
	_ = svc.WatchPlugins()
	err := svc.ScanPlugins()
	if err != nil {
		return err
	}

	// 2: start the runtime, if configured
	svc.mux.Lock()
	_, foundRuntime := svc.plugins[runtimeBin]
	svc.mux.Unlock()
	if foundRuntime {
		// runtime is added and starts first
		_, err = svc._startPlugin(runtimeBin)
		if err != nil {
			slog.Error("Starting runtime failed", "runtimeBin", runtimeBin, "err", err)
			return err
		} else {
			slog.Warn("Runtime started successfully", "runtimeBin", runtimeBin)

		}
	}

	// 3: a connection to the hub is needed to receive requests
	if svc.hc == nil {
		svc.hc, err = connect.ConnectToHub(
			svc.env.ServerURL, svc.env.ClientID, svc.env.CertsDir, "")
		if err != nil {
			err = fmt.Errorf("failed starting launcher service: %w", err)
			return err
		}
	}

	// permissions for using this service
	err = authz.UserSetPermissions(svc.hc, authz.ThingPermissions{
		AgentID: svc.hc.GetClientID(),
		ThingID: launcherapi.ManageServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleManager, authz.ClientRoleAdmin, authz.ClientRoleService},
		Deny:    nil,
	})

	// 4: start listening to action requests
	StartLauncherAgent(svc, svc.hc)

	// 5: autostart the configured 'autostart' plugins
	// Log errors but do not stop the launcher
	for _, name := range svc.cfg.Autostart {
		_, _ = svc._startPlugin(name)
	}

	return err
}

// Stop the launcher and all running plugins
func (svc *LauncherService) Stop() error {
	slog.Info("Stopping launcher service")

	svc.isRunning.Store(false)

	_ = svc.serviceWatcher.Close()

	err := svc.StopAllPlugins(&launcherapi.StopAllPluginsArgs{IncludingRuntime: true})
	return err
}

// WatchPlugins watches the bin and plugins folder for changes and reloads
// This will detect adding new plugins without requiring a restart.
func (svc *LauncherService) WatchPlugins() error {
	svc.serviceWatcher, _ = fsnotify.NewWatcher()
	err := svc.serviceWatcher.Add(svc.env.BinDir)
	if err == nil && svc.env.PluginsDir != "" {
		err = svc.serviceWatcher.Add(svc.env.PluginsDir)
	}
	if err == nil {
		go func() {
			for {
				select {
				case <-svc.done:
					slog.Info("service watcher ended")
					return
				case event := <-svc.serviceWatcher.Events:
					isRunning := svc.isRunning.Load()
					if isRunning {
						slog.Info("watcher event", "event", event)
						_ = svc.ScanPlugins()
					} else {
						slog.Info("service watcher stopped")
						return
					}
				case err := <-svc.serviceWatcher.Errors:
					if err != nil {
						slog.Error("error", "err", err)
					}
				}
			}
		}()

	}
	return err
}

// NewLauncherService returns a new launcher instance for the plugins in the given plugins folder.
// This scans the folder for executables, adds these to the list of available plugins and autostarts plugins
// Logging will be enabled based on LauncherConfig.
//
// The hub client is used to create service accounts if needed.
func NewLauncherService(
	env plugin.AppEnvironment,
	cfg config.LauncherConfig,
	hc clients.IAgent,
) *LauncherService {

	ls := &LauncherService{
		env:     env,
		cfg:     cfg,
		plugins: make(map[string]*launcherapi.PluginInfo),
		cmds:    make([]*exec.Cmd, 0),
		hc:      hc,
	}

	return ls
}
