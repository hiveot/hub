package config

// OWServerConfig contains the plugin configuration
type OWServerConfig struct {
	// Optional loglevel for use, default is warning
	LogLevel string `yaml:"logLevel,omitempty"`

	// ServerURL optional hub address to connect to.
	// Default "" for auto discovery.
	ServerURL string `yaml:"serverUrl,omitempty"`

	// Keyfile with the client's public/private key
	// Generated on first startup if not found.
	// Default is {certsDir}/owserver.key
	//KeyFile string `yaml:"keyFile,omitempty"`

	// OWServerURL optional http://address:port of the EDS OWServer-V2 gateway.
	// Default "" is auto-discover using DNS-SD
	OWServerURL string `yaml:"owServerURL,omitempty"`

	// OWServerLogin and password to the EDS OWserver using Basic Auth.
	OWServerLogin    string `yaml:"owServerLogin,omitempty"`
	OWServerPassword string `yaml:"owServerPassword,omitempty"`

	// TDInterval optional override interval of republishing the full TD, in seconds.
	// Default is on startup and every 24 hours
	TDInterval int `yaml:"tdInterval,omitempty"`

	// PollInterval optional override interval of polling Thing values, in seconds.
	// Default is 60 seconds
	PollInterval int `yaml:"pollInterval,omitempty"`

	// RepublishInterval optional override interval that unmodified Thing values are republished, in seconds.
	// Default is every 60 minutes
	RepublishInterval int `yaml:"republishInterval,omitempty"`
}

// NewConfig returns a OWServerConfig with default values
func NewConfig() *OWServerConfig {
	cfg := OWServerConfig{}

	// ensure valid defaults
	cfg.LogLevel = ""
	// default re-publish the TD docs every 7 days
	cfg.TDInterval = 3600 * 24 * 7
	// poll for value updates every 60 seconds
	cfg.PollInterval = 60
	// republish same-value after 60 minutes
	cfg.RepublishInterval = 60 * 60
	//cfg.AuthTokenFile = "owserver.token"
	//cfg.KeyFile = "owserver.key"
	return &cfg
}
