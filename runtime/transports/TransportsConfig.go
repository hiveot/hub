package transports

import (
	"github.com/hiveot/hub/runtime/transports/discotransport"
)

type ProtocolsConfig struct {

	// Enable the GRPC protocol binding. Default is false.
	EnableGRPC bool `yaml:"enableGRPC,omitempty"`
	// Enable the HTTPS protocol binding. Default is true.
	EnableHTTPS bool `yaml:"enableHTTPS,omitempty"`
	// Enable mDNS discovery. Default is true
	EnableDiscovery bool `yaml:"enableDiscovery,omitempty"`
	// Enable the MQTT protocol binding, default is false.
	EnableMQTT bool `yaml:"enableMQTT,omitempty"`
	// Enable the NATS protocol binding, default is false.
	EnableNATS bool `yaml:"enableNATS,omitempty"`

	// each protocol binding has its own config section
	Discovery      discotransport.DiscoveryConfig          `yaml:"discovery"`
	HttpsTransport httpstransport_old.HttpsTransportConfig `yaml:"httpsBinding"`
	//MqttTransport  *MqttTransportConfig
	//NatsTransport  *NatsTransportConfig
	//GrpcTransport  *GrpcTransportConfig
}

// NewProtocolsConfig creates the default configuration of communication protocols
// This enables https and mdns
func NewProtocolsConfig() ProtocolsConfig {
	cfg := ProtocolsConfig{
		EnableHTTPS:     true,
		EnableDiscovery: true,
		EnableMQTT:      false,
		EnableNATS:      false,
		EnableGRPC:      false,
		Discovery:       discotransport.NewDiscoveryConfig(),
		HttpsTransport:  httpstransport_old.NewHttpsTransportConfig(),
	}
	return cfg
}
