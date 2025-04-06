package launcherapi

// AgentID is the connect ID of the agent connecting to the Hub
const AgentID = "launcher"

// ManageServiceID is the ID of the service as used by the agent
const ManageServiceID = "manage"

// Launcher service management methods
const (
	// ListMethod lists the launch status of plugins
	ListMethod = "list"
	// StartPluginMethod starts a plugin
	StartPluginMethod = "startPlugin"
	// StartAllPluginsMethod starts all known plugins that haven't yet been started
	StartAllPluginsMethod = "startAllPlugins"
	// StopPluginMethod stops a plugin
	StopPluginMethod = "stopPlugin"
	// StopAllPluginsMethod stops all running plugins
	StopAllPluginsMethod = "stopAllPlugins"
)

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
	StartTimeMSE int64

	// Stopped time of the service in msec-since epoc
	StopTimeMSE int64

	// uptime time the service is running in seconds.
	Uptime int
}

type ListArgs struct {
	OnlyRunning bool `json:"onlyRunning"`
}
type ListResp struct {
	PluginInfoList []PluginInfo `json:"info"`
}

type StartPluginArgs struct {
	Name string `json:"name"`
}

type StartPluginResp struct {
	PluginInfo PluginInfo `json:"info"`
}

type StopPluginArgs struct {
	Name string `json:"name"`
}
type StopPluginResp struct {
	PluginInfo PluginInfo `json:"info"`
}

type StopAllPluginsArgs struct {
	// Also stop the runtime
	IncludingRuntime bool `json:"includingRuntime,omitempty"`
}
