package servers

import (
	"github.com/hiveot/hub/lib/discovery"
	"os"
)

const DefaultHttpsPort = 8444
const DefaultHiveotWssPath = "/hiveot/wss"
const DefaultHiveotSsePath = "/hiveot/sse"
const DefaultMqttTcpPort = 8883
const DefaultMqttWssPort = 8884

// const DefaultWotSsePath = "/wot/sse"
const DefaultWotWssPath = "/wot/wss"

type ProtocolsConfig struct {

	// Enable the HiveOT SSE (sse-sc) sub protocol binding. Default is true.
	EnableHiveotSSE bool `yaml:"enableHiveotSSE"`
	// Enable the HiveOT WSS sub protocol binding. Default is true.
	EnableHiveotWSS bool `yaml:"enableHiveotWSS"`

	// Enable the WoT HTTP Basic protocol binding. Default is true.
	EnableWotHTTPBasic bool `yaml:"enableWotHTTPBasic"`
	// Enable the WoT Websocket sub-protocol binding. Default is true.
	EnableWotWSS bool `yaml:"enableWotWSS"`

	// Enable the MQTT protocol binding, default is false.
	//EnableMQTT bool `yaml:"enableMQTT"`

	// Enable mDNS discovery. Default is true
	EnableDiscovery bool `yaml:"enableDiscovery"`

	// Http host interface
	HttpHost string `yaml:"host"`
	// https listening port
	HttpsPort int `yaml:"httpsPort"`
	// WoT SSE subprotocol paths
	//WotSSEPath string `yaml:"wotSsePath"`
	// HiveOT subprotocol prefix
	//HiveotWSSPath string `yaml:"hiveotWssPath"`

	// MQTT host interface
	MqttHost string `yaml:"mqttHost"`
	// MQTT tcp port
	MqttTcpPort int `yaml:"mqttTcpPort"`
	// MQTT websocket port
	MqttWssPort int `yaml:"mqttWssPort"`

	// each protocol binding has its own config section
	Discovery discovery.DiscoveryConfig `yaml:"discovery"`
}

// NewProtocolsConfig creates the default configuration of communication protocols
// This enables https and mdns
func NewProtocolsConfig() ProtocolsConfig {
	hostName, _ := os.Hostname()

	cfg := ProtocolsConfig{
		EnableHiveotSSE:    true,
		EnableHiveotWSS:    true,
		EnableWotHTTPBasic: true,
		EnableWotWSS:       true,
		EnableDiscovery:    true,
		//EnableMQTT:      false, // todo
		//HttpWSSPath:     transports.DefaultWSSPath,
		//HttpSsePath:     transports.DefaultSSEPath,
		//HttpSseScPath:     transports.DefaultSSESCPath,
		HttpHost:    hostName,
		HttpsPort:   DefaultHttpsPort,
		MqttHost:    hostName,
		MqttTcpPort: DefaultMqttTcpPort,
		MqttWssPort: DefaultMqttWssPort,

		Discovery: discovery.NewDiscoveryConfig(hostName),
	}
	return cfg
}
