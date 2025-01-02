package servers

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/discotransport"
	"os"
)

type ProtocolsConfig struct {
	// Enable the GRPC protocol binding. Default is false.
	//EnableGRPC bool `yaml:"enableGRPC,omitempty"`

	// Enable the HTTPS transport binding. Default is true.
	EnableHTTPS bool `yaml:"enableHTTPS"`
	// Enable the SSE-SC sub protocol transport binding. Default is true.
	EnableSSESC bool `yaml:"enableSSESC"`
	// Enable the Websocket sub protocol transport binding. Default is true.
	EnableWSS bool `yaml:"enableWSS"`
	// Enable the MQTT protocol binding, default is false.
	EnableMQTT bool `yaml:"enableMQTT"`

	// Enable mDNS discovery. Default is true
	EnableDiscovery bool `yaml:"enableDiscovery"`

	// Http host interface
	HttpHost string `yaml:"host"`
	// https listening port
	HttpsPort int `yaml:"httpsPort"`
	// SSE subprotocol prefix
	//HttpSSEPath string `yaml:"ssePath"`
	// Websocket subprotocol prefix
	//HttpWSSPath string `yaml:"wssPath"`

	// MQTT host interface
	MqttHost string `yaml:"mqttHost"`
	// MQTT tcp port
	MqttTcpPort int `yaml:"mqttTcpPort"`
	// MQTT websocket port
	MqttWssPort int `yaml:"mqttWssPort"`

	// each protocol binding has its own config section
	Discovery discotransport.DiscoveryConfig `yaml:"discovery"`
}

// NewProtocolsConfig creates the default configuration of communication protocols
// This enables https and mdns
func NewProtocolsConfig() ProtocolsConfig {
	hostName, _ := os.Hostname()

	cfg := ProtocolsConfig{
		EnableHTTPS:     true,
		EnableSSESC:     true,
		EnableWSS:       true,
		EnableDiscovery: true,
		EnableMQTT:      false, // todo
		//HttpWSSPath:     transports.DefaultWSSPath,
		//HttpSsePath:     transports.DefaultSSEPath,
		//HttpSseScPath:     transports.DefaultSSESCPath,
		HttpHost:    hostName,
		HttpsPort:   transports.DefaultHttpsPort,
		MqttHost:    hostName,
		MqttTcpPort: transports.DefaultMqttTcpPort,
		MqttWssPort: transports.DefaultMqttWssPort,

		Discovery: discotransport.NewDiscoveryConfig(),
	}
	return cfg
}
