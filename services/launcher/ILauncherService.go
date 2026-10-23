package launcher

import "github.com/hiveot/hivekit/go/api"

// ThingID, actions, properties, and events
const (

	// LauncherServiceID is the thingID of the launcher service
	LauncherServiceID = "launcher"

	// properties
	PropNrPlugins = "nrPlugins"
	PropNrRunning = "nrRunning"
	// events
	PluginStartedEvent = "pluginStarted"
	PluginStoppedEvent = "pluginStopped"
	// actions
	ListPluginsAction     = "listPlugins"
	StartAllPluginsAction = "startAllPlugins"
	StartPluginAction     = "startPlugin"
	StopAllPluginsAction  = "stopAllPlugins"
	StopPluginAction      = "stopPlugin"
)

// PluginInfo defines a Plugin Information data schema.
//
// PluginInfo contains the running status of a service
type PluginInfo struct {

	// Uptime with Uptime
	//
	// Time the service is running in seconds.
	Uptime int64 `json:"Uptime,omitempty"`

	// Cpu with CPU
	//
	// CPU usage in %. 0 when not running
	Cpu int64 `json:"cpu,omitempty"`

	// ExecPath with Path
	//
	// Path to the service executable
	ExecPath string `json:"execPath,omitempty"`

	// ModifiedTime with Modified
	//
	// Service modified time ISO8601
	ModifiedTime string `json:"modifiedTime,omitempty"`

	// Pid with PID
	//
	// Program PID when started. This remains after stopping.
	Pid int64 `json:"pid,omitempty"`

	// PluginID with Plugin ID
	//
	// ID of the service
	PluginID string `json:"pluginID,omitempty"`

	// Rss with RSS
	//
	// (Resident Set Size) Memory usage in Bytes. 0 when not running.
	Rss int64 `json:"rss,omitempty"`

	// Running with Running
	//
	// Service is currently running
	Running bool `json:"running,omitempty"`

	// Size with Size
	//
	// Binary size of the service in bytes
	Size int64 `json:"size,omitempty"`

	// StartCount with Start Count
	//
	// Number of times the service was restarted
	StartCount int64 `json:"startCount,omitempty"`

	// StartedTime with Start Time
	//
	// Starting time of the service in ISO8601
	StartedTime string `json:"startedTime,omitempty"`

	// Status with Status
	//
	// The last status message received from the process
	Status string `json:"status,omitempty"`

	// StoppedTime with Stopped Time
	StoppedTime string `json:"stoppedTime,omitempty"`
}

// ILauncherService defines the interface of the launcher service
//
// This defines the methods for each of the actions in the TD.
type ILauncherService interface {
	api.IHiveCell

	// ListPlugins List Plugins
	// List available plugins and their status
	ListPlugins(onlyrunning bool) (plugins []PluginInfo, err error)

	// StartAllPlugins Start All
	// Start all available plugins
	StartAllPlugins() error

	// StartPlugin Start A Plugin
	// Start a plugin
	StartPlugin(pluginID string) (pluginStatus PluginInfo, err error)

	// StopAllPlugins Stop All
	// Stop all running plugins
	StopAllPlugins(fullStop bool) error

	// StopPlugin Stop A Plugin
	// Stop a plugin
	StopPlugin(pluginID string) (pluginStatus PluginInfo, err error)
}
