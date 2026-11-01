package config

type IPNetConfig struct {
	// ThingID is this service thingID to publish as. Default is "ipnet"
	ThingID      string `yaml:"thingID,omitempty" `
	LogLevel     string `yaml:"logLevel,omitempty"`
	PollInterval int    `yaml:"pollInterval,omitempty"`

	// also scan common ports
	PortScan bool `yaml:"portScan" json:"portScan"`

	// list of addition ports
	Ports []uint `yaml:"ports"`

	// if sudo is available, then scan ports as root
	ScanAsRoot bool `yaml:"scanAsRoot" json:"scanAsRoot"`
}

// Create a new IPnet binding configuration with defaults
func NewIPNetConfig() *IPNetConfig {
	cfg := IPNetConfig{
		LogLevel:     "warning",
		PollInterval: 3600,
		PortScan:     false,
	}
	return &cfg
}
