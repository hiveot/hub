package launcher

// LauncherConfig holds the configuration of the launcher service
//
// To have the launcher create credentials for its plugins it must have access to then
// authn service. The authn module can be included as part of the application chain,
// be running on a gateway where the launcher is connected to, or linked to it with a client
// module if it runs stand-alone.
// If no authn access is available then set 'CreatePluginCred' to false.
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

	// RuntimeWait is extra wait time in msec after launching the runtime before starting plugins
	// Default is 100msec but this might need to be increased on slow systems.
	RuntimeWait int `yaml:"runtimeWait"`

	// CreatePluginCred creates per-plugin key and token credential files if they don't exist
	// This requires access to 'authn' the authentication service to create an auth token.
	// Either include authn in the request chain or link the launcher to it with a client module.
	// Default is true
	CreatePluginCred bool `yaml:"createPluginCred"`

	// optional override of the logging location; default is ./logs
	// LogsDir string `yaml:"logsDir"`

	// logging level override
	LogLevel string `yaml:"logLevel"`

	// log the launcher to file logs/launcher.log
	LogToFile bool `yaml:"logtofile"`

	// direct stdout of plugins to logfile at logs/{plugin}.log
	LogPlugins bool `yaml:"logPlugins"`

	// provide plugins with the URL of the directory TD, if available.
	// this uses the '--directoryURL' commandline parameter. Default is true.
	ProvideDirectoryURL bool `yaml:"provideDirectoryURL"`

	// provide plugins with the connection URL of the server, if available.
	// this uses the '--serverURL' commandline parameter. Default is true.
	ProvideServerURL bool `yaml:"provideServerURL"`

	// Wait-time in msec after starting plugins to give the time to become ready. Default is 100msec
	StartWait int `yaml:"startWait"`

	// Flag, write the launcher TD to the directory. This requires that the directory
	// is reachable via the launcher sink. Only set this if it is remote controlled or
	// when viewers are allowed to see the launcher and its statuses.
	WriteTD bool `yaml:"writeTD"`
}

// NewLauncherConfig returns a new launcher configuration with defaults
func NewLauncherConfig() *LauncherConfig {
	lc := &LauncherConfig{
		AttachStderr:        true,
		AttachStdout:        false,
		AutoRestart:         false,
		Autostart:           make([]string, 0),
		RuntimeBin:          "",
		CreatePluginCred:    true,
		LogLevel:            "",
		LogToFile:           true,
		LogPlugins:          true,
		ProvideDirectoryURL: true,
		ProvideServerURL:    true,
		StartWait:           100,
		RuntimeWait:         100,
		WriteTD:             true,
	}
	return lc
}
