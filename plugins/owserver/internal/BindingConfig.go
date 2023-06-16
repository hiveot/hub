package internal

import "os"

// DefaultID is the default instance ID of this service. Used to name the configuration file
// and as the publisher ID portion of the Thing ID (zoneID:publisherID:deviceID:deviceType)
const DefaultID = "owserver"

// OWServerConfig contains the plugin configuration
type OWServerConfig struct {
	// ID optional override of the instance ID of the binding in case of multiple instances.
	// Default is 'owserver-hostname'.
	// This must match the id in the client cert
	ID string `yaml:"id"`

	// HubURL optional hub address to connect to
	// Default "" for auto discovery.
	// * "unix://path/to/socket"    for UDS
	// * "tcp://address:4223/"    for TCP/TLS
	// * "wss://address:8444/ws"  for Websocket
	HubURL string `yaml:"hubUrl,omitempty"`

	// OWServerAddress optional http://address:port of the EDS OWServer-V2 gateway.
	// Default "" is auto-discover using DNS-SD
	OWServerAddress string `yaml:"owserverAddress,omitempty"`

	// LoginName and password to the EDS OWserver using Basic Auth.
	LoginName string `yaml:"loginName,omitempty"`
	Password  string `yaml:"password,omitempty"`

	// TDInterval optional override interval of republishing the full TD, in seconds.
	// Default is 12 hours
	TDInterval int `yaml:"tdInterval,omitempty"`

	// PollInterval optional override interval of polling Thing values, in seconds.
	// Default is 60 seconds
	PollInterval int `yaml:"pollInterval,omitempty"`

	// RepublishInterval optional override interval that unmodified Thing values are republished, in seconds.
	// Default is 3600 seconds
	RepublishInterval int `yaml:"republishInterval,omitempty"`
}

// NewConfig returns a OWServerConfig with default values
func NewConfig() OWServerConfig {
	cfg := OWServerConfig{}

	// ensure valid defaults
	hostName, _ := os.Hostname()
	cfg.ID = DefaultID + "-" + hostName
	cfg.TDInterval = 3600 * 12
	cfg.PollInterval = 60
	cfg.RepublishInterval = 3600
	return cfg
}
