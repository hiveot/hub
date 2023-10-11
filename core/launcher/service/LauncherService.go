package service

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/launcher"
	"github.com/hiveot/hub/core/launcher/config"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/hubconnect"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// CoreID The core has a predefined ID
const CoreID = "core"

// LauncherService manages starting and stopping of plugins
// This implements the ILauncher interface
type LauncherService struct {
	// service configuration
	cfg config.LauncherConfig
	env utils.AppEnvironment

	// map of plugin name to running status
	plugins map[string]*launcher.PluginInfo
	// list of started commands in startup order
	cmds []*exec.Cmd

	// hub messaging client
	hc hubclient.IHubClient
	// auth service to generate plugin keys and tokens
	authSvc auth.IAuthnManageClients
	// subscription to receive requests
	mngSub hubclient.ISubscription

	// mutex to keep things safe
	mux sync.Mutex
	// watch plugin folders for updates
	serviceWatcher *fsnotify.Watcher
	// service is running
	isRunning atomic.Bool
	// closing channel
	done chan bool
}

// Add discovered core to svc.plugins
func (svc *LauncherService) addCore() error {
	if svc.cfg.CoreBin != "" {
		corePath := path.Join(svc.env.BinDir, svc.cfg.CoreBin)
		coreInfo, err := os.Stat(corePath)
		if err != nil {
			err = fmt.Errorf("findCore. core in config not found. Path=%s", corePath)
			return err
		}
		pluginInfo, found := svc.plugins[CoreID]
		if found {
			// update existing entry for core
			pluginInfo.ModifiedTime = coreInfo.ModTime().Format(time.RFC3339)
			pluginInfo.Size = coreInfo.Size()
		} else {
			// add new entry for core
			pluginInfo = &launcher.PluginInfo{
				Name:    coreInfo.Name(),
				Path:    corePath,
				Uptime:  0,
				Running: false,
			}
			pluginInfo.ModifiedTime = coreInfo.ModTime().Format(time.RFC3339)
			pluginInfo.Size = coreInfo.Size()
			svc.plugins[CoreID] = pluginInfo
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
		fileInfo, _ := entry.Info()
		size := fileInfo.Size()
		fileMode := fileInfo.Mode()
		isExecutable := fileMode&0100 != 0
		isFile := !entry.IsDir()
		if isFile && isExecutable && size > 0 {
			count++
			pluginInfo, found := svc.plugins[entry.Name()]
			if !found {
				pluginInfo = &launcher.PluginInfo{
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
	slog.Info("found plugins", "count", count, "directory", folder)
	return nil
}

// List all available or just the running plugins and their status
// This returns the list of plugins sorted by name
func (svc *LauncherService) List(onlyRunning bool) ([]launcher.PluginInfo, error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	// get the keys of the plugins to include and sort them
	keys := make([]string, 0, len(svc.plugins))
	for key, val := range svc.plugins {
		if !onlyRunning || val.Running {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	res := make([]launcher.PluginInfo, 0, len(keys))
	for _, key := range keys {
		svcInfo := svc.plugins[key]
		svc.updateStatus(svcInfo)
		res = append(res, *svcInfo)
	}
	return res, nil
}

// ScanPlugins scans the plugin folder for changes and updates the plugins list
func (svc *LauncherService) ScanPlugins() error {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	//// include the core
	//err := svc.addCore()
	//if err != nil {
	//	slog.Error(err.Error())
	//	return err
	//}
	// add plugins
	err := svc.addPlugins(svc.env.PluginsDir)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

// Start the launcher service
// This first starts the core defined in the config, then connects to the hub
// to be able to create auth keys and tokens, and to subscribe to rpc requests.
//
// Call stop to end
func (svc *LauncherService) Start() error {

	svc.isRunning.Store(true)

	// include the core
	if svc.cfg.CoreBin != "" {
		err := svc.addCore()
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

	// 2: start the core, if configured
	svc.mux.Lock()
	_, found := svc.plugins[CoreID]
	svc.mux.Unlock()
	if found {
		// core is added
		_, err = svc.StartPlugin(CoreID)
		if err != nil {
			return err
		}
	}

	// 3: a connection to the message bus is needed
	if svc.hc == nil {
		svc.hc, err = hubconnect.ConnectToHub(
			svc.env.ServerURL, svc.env.ClientID, svc.env.CertsDir, "")
		if err != nil {
			err = fmt.Errorf("failed starting launcher service: %w", err)
			return err
		}
	}

	// the auth service is used to create plugin credentials
	svc.authSvc = authclient.NewAuthClientsClient(svc.hc)

	// start listening to requests
	svc.mngSub, err = svc.hc.SubRPCRequest(launcher.LauncherManageCapability, svc.HandleRequest)

	// 4: autostart the configured 'autostart' plugins
	for _, name := range svc.cfg.Autostart {
		_, err2 := svc.StartPlugin(name)
		if err2 != nil {
			err = err2
		}
	}
	return err
}

// Stop the launcher and all running plugins
func (svc *LauncherService) Stop() error {
	if svc.mngSub != nil {
		svc.mngSub.Unsubscribe()
		svc.mngSub = nil
	}
	svc.isRunning.Store(false)
	return svc.StopAllPlugins()
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
					slog.Error("error", "err", err)
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
// The hub client is intended when an existing message bus is used. If the core is
// started by the launcher then it is ignored.
func NewLauncherService(
	env utils.AppEnvironment,
	cfg config.LauncherConfig,
	hc hubclient.IHubClient,
) *LauncherService {

	ls := &LauncherService{
		env:     env,
		cfg:     cfg,
		plugins: make(map[string]*launcher.PluginInfo),
		cmds:    make([]*exec.Cmd, 0),
		hc:      hc,
	}

	return ls
}
