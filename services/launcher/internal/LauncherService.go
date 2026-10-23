package internal

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/agent"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/launcher"
	"github.com/hiveot/hub/services/launcher/tm"
)

// Use this default path instead of discovery when running locally and no server is configured
const DefaultLocalServerURL = "wss://localhost" + wss.WotWebsocketPath

// LauncherService manages starting and stopping of plugins
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
type LauncherService struct {
	*agent.Agent

	// service configuration
	cfg *launcher.LauncherConfig

	//
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
	isRunning atomic.Bool
	// closing channel
	done chan bool

	// request handler
	adminHandler msg.RequestHandler
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
		if !found {
			// add new entry for runtime
			pluginInfo = launcher.PluginInfo{
				PluginID: runtimeInfo.Name(),
				ExecPath: runtimePath,
				Uptime:   0,
				Running:  false,
			}
			svc.plugins[runtimeBin] = pluginInfo
		}
		pluginInfo.ModifiedTime = utils.FormatUTCMilli(runtimeInfo.ModTime())
		pluginInfo.Size = runtimeInfo.Size()
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
					pluginInfo = launcher.PluginInfo{
						PluginID: entry.Name(),
						ExecPath: path.Join(folder, entry.Name()),
						Uptime:   0,
						Running:  false,
					}
					svc.plugins[pluginInfo.PluginID] = pluginInfo
				}
				pluginInfo.ModifiedTime = fileInfo.ModTime().Format(time.RFC3339)
				pluginInfo.Size = size
			}
		}
	}
	slog.Info("found plugins", "count", count, "directory", folder)
	return nil
}

// ListPlugins lists all available or just the running plugins and their status
// This returns the list of plugins sorted by name
func (svc *LauncherService) ListPlugins(onlyRunning bool) ([]launcher.PluginInfo, error) {
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

// Start the launcher service.
//
// This first starts the runtime defined in the config, then connects to the hub
// to be able to create auth keys and tokens, and to subscribe to rpc requests.
//
// Call stop to end
func (svc *LauncherService) Start() error {
	slog.Info("Starting LauncherService")
	svc.isRunning.Store(true)

	env := svc.env
	cfg := svc.cfg

	// include the runtime
	runtimeBin := cfg.RuntimeBin
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
		pi, err := svc._startPlugin(runtimeBin)
		if err != nil {
			slog.Error("Starting runtime failed", "runtimeBin", runtimeBin, "err", err)
			return err
		} else {
			slog.Warn("Runtime started successfully", "runtimeBin", runtimeBin)

			// a bit of a stopgap for slow systems
			// FIXME: runtime should signal readiness
			time.Sleep(time.Millisecond * time.Duration(cfg.RuntimeWait))

			// some extra time if its still busy
			if pi.Cpu >= 99 {
				time.Sleep(time.Second)
			}
		}
		// since the runtime is launched locally a local connection is the most efficient
		// unless a server address is already configured.
		if env.ServerURL == "" {
			env.ServerURL = DefaultLocalServerURL
		}
	}

	// 3. publish this service TD if connected to a directory or discovery
	if cfg.WriteTD {
		err = svc.WriteTD(string(tm.LauncherTM))
		if err != nil {
			slog.Error("failed to publish the launcher service TD", "err", err.Error())
		}
	}

	// 4. Set permissions for using this service for administrators and managers
	// err = authz.UserSetPermissions(svc.ag.Consumer, authz.ThingPermissions{
	// 	AgentID: svc.ag.GetClientID(),
	// 	ThingID: launcher.AdminServiceID,
	// 	Allow:   []authz.ClientRole{authz.ClientRoleManager, authz.ClientRoleAdmin, authz.ClientRoleService},
	// 	Deny:    nil,
	// })

	// 5: autostart the configured 'autostart' plugins
	// Log errors but do not stop the launcher
	for _, name := range cfg.Autostart {
		_, _ = svc._startPlugin(name)
	}

	return err
}

// Stop the launcher, all running plugins and disconnect
func (svc *LauncherService) Stop() {
	slog.Info("Stopping launcher service")

	svc.isRunning.Store(false)

	_ = svc.serviceWatcher.Close()

	svc.StopAllPlugins(true)
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

// NewLauncherService returns a new launcher service instance for the plugins in the
// given plugins folder.
// This scans the folder for executables, adds these to the list of available plugins and autostarts plugins
// Logging will be enabled based on LauncherConfig.
//
// The sink forwards requests to create service accounts to authn service.
// env is the application environment with bin, certs, log and plugin directories
// cfg is the launcher configuration
func NewLauncherService(
	sink api.IHiveCell, env *api.AppEnvironment, cfg *launcher.LauncherConfig,

	// ag transports.IClientConnection,
) *LauncherService {

	svc := &LauncherService{
		Agent: agent.NewAgent(env.AppID, nil),
		env:   env,
		// serverURL: serverURL,
		cfg:     cfg,
		plugins: make(map[string]launcher.PluginInfo),
		cmds:    make([]*exec.Cmd, 0),
		//ag:        ag,
	}
	// check property api implementation
	var _ launcher.ILauncherService = svc
	var _ api.IHiveCell = svc

	return svc
}
