package protocols

import "github.com/hiveot/hub/runtime/protocols/httpsbinding"

type ProtocolsConfig struct {

	// each protocol binding has its own config section
	HttpsBinding httpsbinding.HttpsBindingConfig `yaml:"httpsBinding"`
	//MqttBinding  *MqttBindingConfig
	//NatsBinding  *NatsBindingConfig
	//GrpcBinding  *GrpcBindingConfig
}

func NewProtocolsConfig() ProtocolsConfig {
	cfg := ProtocolsConfig{
		HttpsBinding: httpsbinding.NewHttpsBindingConfig(),
	}
	return cfg
}
