package transports

import "github.com/hiveot/hub/runtime/transports/httpstransport"

type ProtocolsConfig struct {

	// each protocol binding has its own config section
	HttpsBinding httpstransport.HttpsTransportConfig `yaml:"httpsBinding"`
	//MqttBinding  *MqttBindingConfig
	//NatsBinding  *NatsBindingConfig
	//GrpcBinding  *GrpcBindingConfig
}

func NewProtocolsConfig() ProtocolsConfig {
	cfg := ProtocolsConfig{
		HttpsBinding: httpstransport.NewHttpsTransportConfig(),
	}
	return cfg
}
