// Package launcher with the launcher interface
package launcher

// ServiceName used to connect to this service
const ServiceName = "launcher"

// LauncherManageCapability is the name of the Thing/Capability that handles management requests
const LauncherManageCapability = "manage"

// PluginInfo contains the running status of a service
type PluginInfo struct {
	// CPU usage in %. 0 when not running
	CPU int

	// RSS (Resident Set Size) Memory usage in Bytes. 0 when not running.
	RSS int

	// Service modified time ISO8601
	ModifiedTime string

	// name of the service
	Name string

	// path to service executable
	Path string

	// Program PID when started. This remains after stopping.
	PID int

	// Service is currently running
	Running bool

	// binary size of the service in bytes
	Size int64

	// The last status message received from the process
	Status string

	// Number of times the service was restarted
	StartCount int

	// Starting time of the service in ISO8601
	StartTime string

	// Stopped time of the service in ISO8601
	StopTime string

	// uptime time the service is running in seconds.
	Uptime int
}

const LauncherListReq = "list"

type LauncherListArgs struct {
	OnlyRunning bool `json:"onlyRunning"`
}
type LauncherListResp struct {
	ServiceInfoList []PluginInfo `json:"serviceInfo"`
}

const LauncherStartPluginReq = "startPlugin"

type LauncherStartPluginArgs struct {
	Name string `json:"name"`
}

type LauncherStartPluginResp struct {
	ServiceInfo PluginInfo `json:"info"`
}

const LauncherStartAllPluginsReq = "startAllPlugins"

// StartAll has no arguments

const LauncherStopPluginReq = "stopPlugin"

type LauncherStopPluginArgs struct {
	Name string `json:"name"`
}
type LauncherStopPluginResp struct {
	ServiceInfo PluginInfo `json:"info"`
}

const LauncherStopAllPluginsReq = "stopAllPlugins"

// ILauncher defines the POGS based interface of the launcher service
type ILauncher interface {

	// List services
	List(onlyRunning bool) ([]PluginInfo, error)

	// Start the service and connect to the hub
	// If a hub core is set in config then start the core first.
	Start() error

	// StartAllPlugins starts all enabled plugins
	// This returns the error from the last service that could not be started
	StartAllPlugins() error

	// StartPlugin start the plugin with the given name.
	// This creates a key and token for the plugin to use to authenticate.
	// The pluginID is the binary name. This allows to run instances of the same plugin on multiple hosts
	// simply by renaming the binary.
	// If the plugin is already running this does nothing
	StartPlugin(name string) (PluginInfo, error)

	// Stop the service and disconnect from the hub
	// If the hub core was started, stop it last.
	Stop() error

	// StopAllPlugins running plugins
	StopAllPlugins() error

	// StopPlugin stops a running plugin
	StopPlugin(name string) (PluginInfo, error)
}
