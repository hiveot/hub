package servers

import (
	"os"

	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
)

const (
	DefaultMqttTcpPort = 8883
	DefaultMqttWssPort = 8884
)

type ProtocolsConfig struct {

	// Enable the HiveOT HTTP/Authentication endpoint. Default is true.
	EnableHiveotAuth bool `yaml:"enableHiveotAuth"`
	// Enable the HiveOT HTTP/SSE (sse-sc) sub protocol binding. Default is true.
	EnableHiveotSSE bool `yaml:"enableHiveotSSE"`
	// Enable the HTTP/WSS sub protocol binding. Default is true.
	EnableWSS bool `yaml:"enableWSS"`

	// Include forms in each affordance to meet specifications.
	// Note that this is not useful when talking to the Hub as all affordances of
	// all digital twin TD have the same forms with the hub as endpoint.
	// This results in massive duplication of TD content and increases the TD by a lot.
	// Don't use forms unless you really need it.
	IncludeForms bool `yaml:"includeForms,omitempty"`

	// Enable the MQTT protocol binding, default is false.
	//EnableMQTT bool `yaml:"enableMQTT"`

	// Enable mDNS discovery. Default is true.
	EnableDiscovery bool `yaml:"enableDiscovery"`
	// The service discovery instance. The default is the hostname
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
	hostName, _ := os.Hostname()

	cfg := ProtocolsConfig{
		DirectoryTDPath:  discoserver.DefaultHttpGetDirectoryTDPath,
		EnableHiveotAuth: true,
		EnableHiveotSSE:  true,
		EnableWSS:        true,
		EnableDiscovery:  true,
		IncludeForms:     true, // for interoperability
		InstanceName:     hostName,
		HttpHost:         "",
		HttpsPort:        httpserver.DefaultHttpsPort,
		MqttHost:         hostName,
		MqttTcpPort:      DefaultMqttTcpPort,
		MqttWssPort:      DefaultMqttWssPort,
	}
	return cfg
}
