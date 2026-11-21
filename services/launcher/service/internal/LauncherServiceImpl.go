package internal

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	certs_client "github.com/hiveot/hivekit/go/cells/certs/client"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	"github.com/hiveot/hub/services/launcher"
)

// Use this default path instead of discovery when running locally and no server is configured
const DefaultLocalServerURL = "wss://localhost" + wss.WotWebsocketPath

// LauncherServiceImpl manages starting and stopping of plugins
// This implements the ILauncher interface and is an Agent module.
//
// Note that this module does not create server connections. It is a module that should
// be linked to transport cells.
//
// A: When used with a gateway, link it to a transport module to make it reachable:
// > Launcher <=> {transport client} <=> {gateway runtime}
//
// B: When used stand-alone it is just another Thing on the network:
// > {transport server} <=> Launcher <=> Discovery
//
// C: Combine A and B, direct access to the launcher while linking to a gateway server
// > {gRPC server} <=> Launcher <=> {transport client} <=> {gateway runtime}
type LauncherServiceImpl struct {
	*thing.ExposedThing

	// service configuration
	cfg *launcher.LauncherConfig

	certsClient *certs_client.CertsClient
	authnAdmin  *authn_client.AuthnAdminClient

	// environment to use for starting plugins
	env *api.AppEnvironment

	// map of plugin name to running status
	plugins map[string]launcher.PluginInfo

	// list of started commands in startup order
	cmds []*exec.Cmd

	// mutex to keep things safe
	mux sync.Mutex
	// watch plugin folders for updates
	serviceWatcher *fsnotify.Watcher
	// service is running
	// isRunning atomic.Bool
	// closing channel
	done chan bool
}

// Add discovered runtime to svc.plugins
func (svc *LauncherServiceImpl) addRuntime(runtimeBin string) error {
	if runtimeBin != "" {
		runtimePath := path.Join(svc.env.BinDir, runtimeBin)
		_, err := os.Stat(runtimePath)
		if err != nil {
			err = fmt.Errorf("addRuntime. runtime in config not found. Path=%s", runtimePath)
			return err
		}
		svc.addPlugin(runtimePath)
	}
	return nil
}

// Add a newly discovered executable plugin to svc.plugins
// If the plugin is already know, only update its size and timestamp
func (svc *LauncherServiceImpl) addPlugin(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("addPlugin: Unable to read plugin info. Skipped", "filepath", filePath, "err", err.Error())
		return err
	}
	if fileInfo.IsDir() {
		// ignore directory
		return nil
	}
	fileMode := fileInfo.Mode()
	isExecutable := fileMode&0100 != 0
	if !isExecutable {
		// ignore non executable
		slog.Info("addPlugin: Ignored non-executable.", "file", filePath)
		return nil
	}
	size := fileInfo.Size()
	if size == 0 {
		slog.Info("addPlugin: Ignored empty file.", "file", filePath)
		return nil
	}

	pluginInfo, found := svc.plugins[fileInfo.Name()]
	if !found {
		pluginInfo = launcher.PluginInfo{
			PluginID:     fileInfo.Name(),
			ExecPath:     filePath,
			ModifiedTime: fileInfo.ModTime().Format(time.RFC3339),
			Size:         size,
			Uptime:       0,
			Running:      false,
		}
		svc.plugins[pluginInfo.PluginID] = pluginInfo
		slog.Info("addPlugin: found new plugin", "file", filePath)
	} else {
		// updated
		pluginInfo.ModifiedTime = fileInfo.ModTime().Format(time.RFC3339)
		pluginInfo.Size = size
	}

	return nil
}

// Add newly discovered executable plugins to svc.plugins
// If the service is already know, only update its size and timestamp
func (svc *LauncherServiceImpl) addPlugins(folder string) error {
	count := 0
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {

		filePath := filepath.Join(folder, entry.Name())
		svc.addPlugin(filePath)
	}
	if count > 0 {
		slog.Info("addPlugins: found new plugins", "count", count, "directory", folder)
	}
	return nil
}

// ListPlugins lists all available or just the running plugins and their status
// This returns the list of plugins sorted by name
func (svc *LauncherServiceImpl) ListPlugins(onlyRunning bool) ([]launcher.PluginInfo, error) {
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

	infoList := make([]launcher.PluginInfo, 0, len(keys))
	for _, key := range keys {
		svcInfo := svc.plugins[key]
		svc.updateStatus(&svcInfo)
		infoList = append(infoList, svcInfo)
	}
	return infoList, nil
}

// Remove a deleted plugin
func (svc *LauncherServiceImpl) removePlugin(path string) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	filename := filepath.Base(path)
	_, found := svc.plugins[filename]
	if found {
		delete(svc.plugins, filename)
		slog.Info("removePlugin", "path", path)
	}

}

// ScanPlugins scans the plugin folder for changes and updates the plugins list
func (svc *LauncherServiceImpl) ScanPlugins() error {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	err := svc.addPlugins(svc.env.PluginsDir)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

// Start the launcher service.
//
// This first starts the runtime defined in the config, then connects to the hub
// to be able to create auth keys and tokens, and to subscribe to rpc requests.
//
// Call stop to end
func (svc *LauncherServiceImpl) Start() {
	slog.Info("Starting LauncherService")

	// 1: start the runtime, if configured
	svc._startRuntime()

	// 2. publish this service TD if connected to a directory or discovery
	if svc.cfg.WriteTD {
		tdoc, err := wot.UnmarshalTD(string(launcher.LauncherTD))
		if err == nil {
			tdoc.ID = svc.GetID()
			tdoc.SetType(launcher.LauncherServiceCellType)
			err = svc.WriteTD(tdoc)
		}
		if err != nil {
			slog.Error("failed to publish the launcher service TD", "err", err.Error())
		}
	}
	nrPlugins := len(svc.plugins)
	svc.PubProperty(launcher.PropNrPlugins, nrPlugins, false)

	// FIXME:
	// 3. Set permissions for using this service for administrators and managers
	// err = authz.UserSetPermissions(svc.ag.Consumer, authz.ThingPermissions{
	// 	AgentID: svc.ag.GetClientID(),
	// 	ThingID: launcher.AdminServiceID,
	// 	Allow:   []authz.ClientRole{authz.ClientRoleManager, authz.ClientRoleAdmin, authz.ClientRoleService},
	// 	Deny:    nil,
	// })

	// // 4. Set permissions for using this service for administrators and managers
	// // err = authz.UserSetPermissions(svc.ag.Consumer, authz.ThingPermissions{
	// // 	AgentID: svc.ag.GetClientID(),
	// // 	ThingID: launcher.AdminServiceID,
	// // 	Allow:   []authz.ClientRole{authz.ClientRoleManager, authz.ClientRoleAdmin, authz.ClientRoleService},
	// // 	Deny:    nil,
	// // })

	// 5: autostart the configured 'autostart' plugins
	// Log errors but do not stop the launcher
	for _, name := range svc.cfg.Autostart {
		_, _ = svc._startPlugin(name)
	}

}

// Stop the launcher, all running plugins and disconnect
func (svc *LauncherServiceImpl) Stop() {
	slog.Info("Stopping launcher service")

	// svc.isRunning.Store(false)

	_ = svc.serviceWatcher.Close()

	svc.StopAllPlugins(true)
}

// WatchPlugins watches the bin and plugins folder for changes and reloads
// This will detect adding new plugins without requiring a restart.
func (svc *LauncherServiceImpl) WatchPlugins() error {
	svc.serviceWatcher, _ = fsnotify.NewWatcher()
	var timers = make(map[string]*time.Timer)
	var debounceMux sync.Mutex
	debounceDuration := time.Second

	// err := svc.serviceWatcher.Add(svc.env.BinDir)
	// if err == nil {
	err := svc.serviceWatcher.Add(svc.env.PluginsDir)
	// }
	if err == nil {
		go func() {
			for {
				select {
				// case <-svc.done:
				// 	slog.Info("service watcher ended")
				// 	return
				case event, ok := <-svc.serviceWatcher.Events:
					if !ok {
						slog.Info("service watcher ended")
						return
					}
					// debounce multiple events. Wait until write has stopped
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
						debounceMux.Lock()
						if t, exists := timers[event.Name]; exists {
							t.Stop()
						}
						timers[event.Name] = time.AfterFunc(debounceDuration, func() {
							debounceMux.Lock()
							delete(timers, event.Name)
							debounceMux.Unlock()
							svc.addPlugin(event.Name)
							slog.Info("WatchPlugins file updated", "event", event.Name)
							// _ = svc.ScanPlugins()
						})
						debounceMux.Unlock()
					}
					if event.Has(fsnotify.Remove) {
						slog.Info("WatchPlugins file removed", "event", event.Name)
						svc.removePlugin(event.Name)
					}
					// isRunning := svc.isRunning.Load()
					// if isRunning {
					// } else {
					// 	slog.Info("service watcher stopped")
					// 	return
					// }
				case err := <-svc.serviceWatcher.Errors:
					if err != nil {
						slog.Error("WatchPlugins error", "err", err)
					}
				}
			}
		}()

	}
	return err
}

// NewLauncherServiceImpl returns a new launcher service instance for the plugins in the
// given plugins folder.
// This scans the folder for executables, adds these to the list of available plugins and autostarts plugins
// Logging will be enabled based on LauncherConfig.
//
// env is the application environment with bin, certs, log and plugin directories
// cfg is the launcher configuration
func NewLauncherServiceImpl(env *api.AppEnvironment, cfg *launcher.LauncherConfig,
) (*LauncherServiceImpl, error) {

	clientID := env.ClientID
	if clientID == "" {
		clientID = launcher.LauncherServiceCellType
	}

	svc := &LauncherServiceImpl{
		ExposedThing: thing.NewExposedThing(clientID, nil),
		env:          env,
		cfg:          cfg,
		plugins:      make(map[string]launcher.PluginInfo),
		cmds:         make([]*exec.Cmd, 0),
	}
	// both authn and certs client send their request via this service. Notifications are no used.
	svc.authnAdmin = authn_client.NewAuthnAdminClient("", nil)
	svc.certsClient = certs_client.NewCertsClient("", nil)
	svc.authnAdmin.SetRequestSink(svc)
	svc.certsClient.SetRequestSink(svc)

	// 1: determine the inventory of plugins
	// include the runtime
	runtimeBin := cfg.RuntimeBin
	if runtimeBin != "" {
		err := svc.addRuntime(runtimeBin)
		if err != nil {
			slog.Error(err.Error())
			return nil, err
		}
	}
	_ = svc.WatchPlugins()
	err := svc.ScanPlugins()
	if err != nil {
		return nil, err
	}

	// check property api implementation
	var _ launcher.ILauncherService = svc
	var _ api.IHiveCell = svc

	return svc, nil
}
