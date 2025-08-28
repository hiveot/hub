package service

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
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients"
	"github.com/hiveot/hub/messaging/servers/wssserver"
	authz "github.com/hiveot/hub/runtime/authz/api"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	launcher "github.com/hiveot/hub/services/launcher/api"
	"github.com/hiveot/hub/services/launcher/config"
)

// Use this default path instead of discovery when running locally and no server is configured
const DefaultLocalServerURL = "wss://localhost" + wssserver.DefaultWssPath

// LauncherService manages starting and stopping of plugins
// This implements the ILauncher interface
type LauncherService struct {
	// service configuration
	cfg config.LauncherConfig
	//env plugin.AppEnvironment
	// server to use or "" for auto discovery
	serverURL string
	// URL with the directory TD
	directoryURL string

	clientID string

	// directories for launching plugins and obtaining certificates and keys
	binDir     string
	certsDir   string
	pluginsDir string

	// map of plugin name to running status
	plugins map[string]launcher.PluginInfo
	// list of started commands in startup order
	cmds []*exec.Cmd

	// agent messaging client
	ag *messaging.Agent

	// mutex to keep things safe
	mux sync.Mutex
	// watch plugin folders for updates
	serviceWatcher *fsnotify.Watcher
	// service is running
	isRunning atomic.Bool
	// closing channel
	done chan bool

	// request handler
	adminHandler messaging.RequestHandler
}

// Add discovered runtime to svc.plugins
func (svc *LauncherService) addRuntime(runtimeBin string) error {
	if runtimeBin != "" {
		runtimePath := path.Join(svc.binDir, runtimeBin)
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
func (svc *LauncherService) ListPlugins(senderID string, onlyRunning bool) ([]launcher.PluginInfo, error) {
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

	err := svc.addPlugins(svc.pluginsDir)
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
		if svc.serverURL == "" {
			svc.serverURL = DefaultLocalServerURL
		}
	}

	// 3: a connection to the hub is needed to receive requests
	// this was delayed until after the runtime is up and running
	// if a local runtime is started then the plugins can use localhost
	if svc.ag == nil {
		cc, token, _, err := clients.ConnectWithTokenFile(
			svc.clientID, svc.certsDir, svc.serverURL, 0)
		_ = token
		if err == nil {
			svc.ag = messaging.NewAgent(cc, nil, nil, nil, nil, 0)
		} else {
			err = fmt.Errorf("failed starting launcher service: %w", err)
			return err
		}
	}

	// publish this service TD
	err = digitwin.ThingDirectoryUpdateThing(svc.ag.Consumer, launcher.AdminTD)
	if err != nil {
		slog.Error("failed to publish the launcher service TD", "err", err.Error())
	}

	// permissions for using this service for administrators and managers
	err = authz.UserSetPermissions(svc.ag.Consumer, authz.ThingPermissions{
		AgentID: svc.ag.GetClientID(),
		ThingID: launcher.AdminServiceID,
		Allow:   []authz.ClientRole{authz.ClientRoleManager, authz.ClientRoleAdmin, authz.ClientRoleService},
		Deny:    nil,
	})

	// 4: start listening to action requests
	adminHandler := launcher.NewHandleAdminRequest(svc)
	svc.ag.SetRequestHandler(adminHandler)

	//StartLauncherAgent(svc, svc.ag)

	// 5: autostart the configured 'autostart' plugins
	// Log errors but do not stop the launcher
	for _, name := range svc.cfg.Autostart {
		_, _ = svc._startPlugin(name)
	}

	return err
}

// Stop the launcher, all running plugins and disconnect
func (svc *LauncherService) Stop() error {
	slog.Info("Stopping launcher service")

	svc.isRunning.Store(false)

	_ = svc.serviceWatcher.Close()

	err := svc.StopAllPlugins(svc.clientID, true)
	if svc.ag != nil {
		svc.ag.Disconnect()
	}
	return err
}

// WatchPlugins watches the bin and plugins folder for changes and reloads
// This will detect adding new plugins without requiring a restart.
func (svc *LauncherService) WatchPlugins() error {
	svc.serviceWatcher, _ = fsnotify.NewWatcher()
	err := svc.serviceWatcher.Add(svc.binDir)
	if err == nil && svc.pluginsDir != "" {
		err = svc.serviceWatcher.Add(svc.pluginsDir)
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
//	serverURL to connect to once runtime is started
//	clientID of this service
//	binDir with location of the runtime
//	pluginsDir with location of plugins
//	certsDir with location of caCert and service certs/keys
//
// The hub client is used to create service accounts if needed.
func NewLauncherService(
	serverURL string,
	clientID string,
	binDir string,
	pluginsDir string,
	certsDir string,
	cfg config.LauncherConfig,
	// ag transports.IClientConnection,
) *LauncherService {

	ls := &LauncherService{
		pluginsDir: pluginsDir,
		binDir:     binDir,
		certsDir:   certsDir,
		serverURL:  serverURL,
		clientID:   clientID,
		cfg:        cfg,
		plugins:    make(map[string]launcher.PluginInfo),
		cmds:       make([]*exec.Cmd, 0),
		//ag:        ag,
	}
	// check property api implementation
	_ = launcher.IAdminService(ls)

	return ls
}
