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

	// log the launcher to file logs/launcher.log
	LogLauncher bool `yaml:"loglauncher"`

	// logging level. default is warning
	LogLevel string `yaml:"loglevel"`

	// direct stdout of services to logfile at logs/{service}.log
	LogServices bool `yaml:"logservices"`
}

// NewLauncherConfig returns a new launcher configuration with defaults
func NewLauncherConfig() LauncherConfig {
	lc := LauncherConfig{
		AttachStdout: false,
		AttachStderr: true,
		AutoRestart:  false,
		Autostart:    make([]string, 0),
		LogLevel:     "warning",
		LogLauncher:  true,
		LogServices:  true,
	}
	return lc
}
