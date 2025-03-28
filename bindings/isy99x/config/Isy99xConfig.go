package config

// DefaultPollIntervalSec for polling the gateway values
const DefaultPollIntervalSec = 1 * 60

// DefaultTDIntervalSec for republishing the node TDs
const DefaultTDIntervalSec = 24 * 3600 * 3

// Isy99xConfig with overridable settings
type Isy99xConfig struct {
	IsyAddress string `yaml:"isyAddr"`  // gateway IP address
	LoginName  string `yaml:"login"`    // gateway login
	Password   string `yaml:"password"` // gateway password

	// LogLevel, optional. Default from environment
	LogLevel string `yaml:"logLevel,omitempty"`

	// Interval in seconds to poll and publish ISY99x values. 0 to not poll.
	PollInterval int `yaml:"pollInterval,omitempty"`
	// Interval in seconds to read and publish node TD documents
	TDInterval int `yaml:"tdInterval,omitempty"`

	// RepublishInterval interval that unmodified Thing values are republished, in seconds.
	// Default is every 60 minutes
	RepublishInterval int `yaml:"republishInterval,omitempty"`
}

func NewIsy99xConfig() *Isy99xConfig {
	cfg := &Isy99xConfig{
		IsyAddress:   "", // use auto config
		LoginName:    "",
		Password:     "",
		LogLevel:     "warn",
		PollInterval: DefaultPollIntervalSec,
		// interval of republishing node TD documents. Default is 24 hours
		TDInterval: DefaultTDIntervalSec,
		// republish unchanged-values after 60 minutes
		RepublishInterval: 60 * 60,
	}
	return cfg
}
