package servers

import (
	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
)

const (
	DefaultMqttTcpPort        = 8883
	DefaultMqttWssPort        = 8884
	DefaultInstanceNameHiveot = "hiveot"
)

type ProtocolsConfig struct {

	// Enable the HiveOT HTTP/Authentication endpoint. Default is true.
	EnableHiveotAuth bool `yaml:"enableHiveotAuth"`
	// Enable the HiveOT HTTP/SSE (sse-sc) sub protocol binding. Default is true.
	EnableHiveotSSE bool `yaml:"enableHiveotSSE"`
	// Enable the HiveOT HTTP/WSS sub protocol binding. Default is true.
	EnableHiveotWSS bool `yaml:"enableHiveotWSS"`
	// Enable the WoT HTTP/Websocket sub-protocol binding. Default is true.
	EnableWotWSS bool `yaml:"enableWotWSS"`

	// Enable the MQTT protocol binding, default is false.
	//EnableMQTT bool `yaml:"enableMQTT"`

	// Enable mDNS discovery. Default is true.
	// The DiscoveryTDPath must be resolved by the http server
	EnableDiscovery bool `yaml:"enableDiscovery"`
	// The service discovery instance. The default is 'hiveot'
	InstanceName string `yaml:"instanceName"`

	// DirectoryTDPath contains the HTTP path to read the digitwin directory TD
	// Defaults to "/.well-known/wot" as per spec
	// This is published by discovery and served by the http server.
	DirectoryTDPath string `json:"directoryTDPath"`

	// Server hostname used in http
	HttpHost string `yaml:"host"`
	// https listening port
	HttpsPort int `yaml:"httpsPort"`

	// HiveOT websocket subprotocol connection path
	// The full URL is included in discovery record parameters
	// with this URL no forms are needed to connect to the hub
	HiveotWSSPath string `yaml:"hiveotWssPath"`
	// HiveOT sse subprotocol connection path
	// The full URL is included in discovery record parameters
	// with this URL no forms are needed to connect to the hub
	HiveotSSEPath string `yaml:"hiveotSsePath"`
	// WoT WSS subprotocol connection path
	// The full URL is included in discovery record parameters
	// with this URL no forms are needed to connect to the hub
	WotWSSPath string `yaml:"wotWssPath"`

	// MQTT host interface
	MqttHost string `yaml:"mqttHost"`
	// MQTT tcp port
	MqttTcpPort int `yaml:"mqttTcpPort"`
	// MQTT websocket port
	MqttWssPort int `yaml:"mqttWssPort"`
}

// NewProtocolsConfig creates the default configuration of communication protocols
// This enables https and mdns
func NewProtocolsConfig() ProtocolsConfig {
	//hostName, _ := os.Hostname()
	hostName := "" // listen on all interfaces

	cfg := ProtocolsConfig{
		DirectoryTDPath:  discoserver.DefaultHttpGetDirectoryTDPath,
		EnableHiveotAuth: true,
		EnableHiveotSSE:  true,
		EnableHiveotWSS:  true,
		EnableWotWSS:     true,
		EnableDiscovery:  true,
		InstanceName:     DefaultInstanceNameHiveot,
		HttpHost:         hostName,
		HttpsPort:        httpserver.DefaultHttpsPort,
		MqttHost:         hostName,
		MqttTcpPort:      DefaultMqttTcpPort,
		MqttWssPort:      DefaultMqttWssPort,
	}
	return cfg
}
