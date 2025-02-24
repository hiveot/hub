package config

// OWServerConfig contains the plugin configuration
// FIXME: use these as TD configuration properties?
// FIXME: load them from the digital twin on reconnect?
// FIXME: hub (re)connection reinitializes the service configuration
//
// how independent from the hub should this be?
//
//	use as wot device? it would need to offer a directory and auth
//
// where to store TDs?  - need a directory
// where to store auth creds - need an auth store
// where to store config/state - need a state store
//
//	why not load config from digitwin on reconnect
//	+support offline updates?
//	-cant function without network
//		services dont function anyways
//	+ a reconnect is a restart (helps recovery)
//	do services hold history? - nothing stopping them from doing so
type OWServerConfig struct {
	// Optional loglevel for use, default is warning
	LogLevel string `yaml:"logLevel,omitempty"`

	// HubURL optional hub address to connect to.
	// Default "" for auto discovery.
	HubURL string `yaml:"hubUrl,omitempty"`

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

	// RepublishInterval interval that unmodified Thing values are republished, in seconds.
	// Default is every 60 minutes
	RepublishInterval int `yaml:"republishInterval,omitempty"`
}

// NewConfig returns a OWServerConfig with default values
func NewConfig() *OWServerConfig {
	cfg := OWServerConfig{}

	// ensure valid defaults
	cfg.LogLevel = ""
	// re-publish node TDs docs every 3 days
	cfg.TDInterval = 3600 * 24 * 3
	// poll for value updates every 60 seconds
	cfg.PollInterval = 60
	// republish same-value after 60 minutes
	cfg.RepublishInterval = 60 * 60
	return &cfg
}
