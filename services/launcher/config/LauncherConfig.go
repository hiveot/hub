package config

// LauncherConfig holds the configuration of the launcher service
type LauncherConfig struct {
	// Attach to service stderr
	AttachStderr bool `yaml:"attachstderr"`

	// Attach to service stdout
	AttachStdout bool `yaml:"attachstdout"`

	// Automatically restart services when they stop with error
	AutoRestart bool `yaml:"autorestart"`

	// List of services to automatically start in launch order
	Autostart []string `yaml:"autostart"`

	// CoreBin to run on startup, if any. Use mqttcore or natscore.
	// Default is not to launch a core.
	CoreBin string `yaml:"corebin"`

	// CreatePluginCred creates per-plugin key and token credential files if they don't exist
	// Default is true
	CreatePluginCred bool `yaml:"createPluginCred"`

	// logging level. default is warning
	LogLevel string `yaml:"logLevel"`

	// log the launcher to file logs/launcher.log
	LogToFile bool `yaml:"logtofile"`

	// direct stdout of plugins to logfile at logs/{plugin}.log
	LogPlugins bool `yaml:"logplugins"`
}

// NewLauncherConfig returns a new launcher configuration with defaults
func NewLauncherConfig() LauncherConfig {
	lc := LauncherConfig{
		AttachStderr:     true,
		AttachStdout:     false,
		AutoRestart:      false,
		Autostart:        make([]string, 0),
		CoreBin:          "",
		CreatePluginCred: true,
		LogLevel:         "warning",
		LogToFile:        true,
		LogPlugins:       true,
	}
	return lc
}
