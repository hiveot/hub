package config

type IPNetConfig struct {
	LogLevel     string `yaml:"logLevel" json:"logLevel"`
	PollInterval int    `yaml:"pollInterval" json:"pollInterval"`

	// also scan common ports
	PortScan bool `yaml:"portScan" json:"portScan"`

	// if sudo is available, then scan ports as root
	ScanAsRoot bool `yaml:"scanAsRoot" json:"scanAsRoot"`
}

func NewIPNetConfig() *IPNetConfig {
	cfg := IPNetConfig{
		LogLevel:     "warning",
		PollInterval: 3600,
		PortScan:     false,
	}
	return &cfg
}
