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

	// RuntimeBin (file) to run on startup, if any.
	// Default is not to launch a runtime.
	RuntimeBin string `yaml:"runtime"`

	// CreatePluginCred creates per-plugin key and token credential files if they don't exist
	// Default is true
	CreatePluginCred bool `yaml:"createPluginCred"`

	// optional override of the logging location; default is ./logs
	LogsDir string `yaml:"logsDir"`

	// logging level. default is warning
	LogLevel string `yaml:"logLevel"`

	// log the launcher to file logs/launcher.log
	LogToFile bool `yaml:"logtofile"`

	// direct stdout of plugins to logfile at logs/{plugin}.log
	LogPlugins bool `yaml:"logPlugins"`

	// provide plugins with the URL of the directory TD, as used by the launcher itself.
	// this uses the '--directoryURL' commandline parameter. Default is true.
	ProvideDirectoryURL bool `yaml:"provideDirectoryURL"`

	// provide plugins with the connection URL of the server, as used by the launcher itself.
	// this uses the '--serverURL' commandline parameter. Default is true.
	ProvideServerURL bool `yaml:"provideServerURL"`
}

// NewLauncherConfig returns a new launcher configuration with defaults
func NewLauncherConfig() LauncherConfig {
	lc := LauncherConfig{
		AttachStderr:        true,
		AttachStdout:        false,
		AutoRestart:         false,
		Autostart:           make([]string, 0),
		RuntimeBin:          "",
		CreatePluginCred:    true,
		LogLevel:            "warning",
		LogToFile:           true,
		LogPlugins:          true,
		ProvideDirectoryURL: true,
		ProvideServerURL:    true,
	}
	return lc
}
