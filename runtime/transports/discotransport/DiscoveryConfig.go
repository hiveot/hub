package discotransport

// DiscoveryConfig HiveOT Hub discovery configuration
type DiscoveryConfig struct {
	// HiveOT instance ID. Default is "hiveot"
	InstanceID string `yaml:"serviceID,omitempty"`
	// HiveOT service ID. Default is "hiveot"
	ServiceID string `yaml:"serviceID,omitempty"`
}

func NewDiscoveryConfig() DiscoveryConfig {
	cfg := DiscoveryConfig{
		InstanceID: "hiveot",
		ServiceID:  "hiveot",
	}
	return cfg
}
