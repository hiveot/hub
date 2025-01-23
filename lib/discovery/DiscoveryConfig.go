package discovery

// DiscoveryConfig HiveOT Hub discovery configuration
type DiscoveryConfig struct {
	// HiveOT instance ID. Default is "hiveot"
	InstanceID string `yaml:"serviceID,omitempty"`
	// HiveOT service ID. Default is "hiveot"
	ServiceID string `yaml:"serviceID,omitempty"`
	//
	// hostname or address of the server
	ServerAddr string `yaml:"serverAddr"`
	// primary port (https)
	ServerPort int `yaml:"port"`

	// connection URL for hiveot SSE-SC protocol
	HiveotSseURL string `yaml:"hiveotSseURL,omitempty"`
	// connection URL for the hiveot websocket protocol
	HiveotWssURL string `yaml:"hiveotWssURL,omitempty"`
	// connection URL for the WoT http-basic protocol
	WotHttpBasicURL string `yaml:"wotHttpBasicURL,omitempty"`
	// connection URL for the WoT websocket protocol
	WotWssURL string `yaml:"wotWssURL,omitempty"`

	//// connection URL for the mqtt over websocket protocol
	//MqttWssURL string `yaml:"mqttWssURL,omitempty"`
	//// connection URL for the mqtt over tcp protocol
	//MqttTcpURL string `yaml:"mqttTcpURL,omitempty"`
}

func NewDiscoveryConfig(serverAddr string) DiscoveryConfig {

	cfg := DiscoveryConfig{
		InstanceID: "hiveot",
		ServiceID:  "hiveot",
		ServerAddr: serverAddr,
	}

	return cfg
}
