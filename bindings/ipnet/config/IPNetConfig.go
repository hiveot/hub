package config

type IPNetConfig struct {
	// AgentID is this service thingID to publish as. Default is "ipnet"
	AgentID      string `yaml:"agentID"`
	LogLevel     string `yaml:"logLevel"`
	PollInterval int    `yaml:"pollInterval"`

	// also scan common ports
	PortScan bool `yaml:"portScan" json:"portScan"`

	// if sudo is available, then scan ports as root
	ScanAsRoot bool `yaml:"scanAsRoot" json:"scanAsRoot"`
}

func NewIPNetConfig() *IPNetConfig {
	cfg := IPNetConfig{
		AgentID:      "ipnet",
		LogLevel:     "warning",
		PollInterval: 3600,
		PortScan:     false,
	}
	return &cfg
}
