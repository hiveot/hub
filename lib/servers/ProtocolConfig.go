package servers

import (
	"os"

	"github.com/hiveot/hub/lib/servers/discoserver"
	"github.com/hiveot/hub/lib/servers/hiveotsseserver"
	"github.com/hiveot/hub/lib/servers/httpbasic"
	"github.com/hiveot/hub/lib/servers/wssserver"
)

const (
	DefaultHttpsPort   = 8444
	DefaultMqttTcpPort = 8883
	DefaultMqttWssPort = 8884
)

type ProtocolsConfig struct {

	// Enable mDNS discovery. Default is true.
	EnableDiscovery bool `yaml:"enableDiscovery,omitempty"`
	// Enable the HiveOT HTTP/Authentication endpoint. Default is true.
	EnableHiveotAuth bool `yaml:"enableHiveotAuth,omitempty"`
	// Enable the HiveOT HTTP/SSE (sse-sc) sub protocol binding. Default is true.
	EnableHiveotSSE bool `yaml:"enableHiveotSSE,omitempty"`
	// EnableHttpBasic. Default is true.
	EnableHttpBasic bool `yaml:"enableHttpBasic,omitempty"`
	// EnableHttpStatic enables a protected static file server, default is false.
	EnableHttpStatic bool `yaml:"enableHttStatic,omitempty"`
	// Enable the HTTP/WSS sub protocol binding. Default is true.
	EnableWSS bool `yaml:"enableWSS,omitempty"`

	// Include forms in each affordance to meet specifications.
	// Note that this is not useful when talking to the Hub as all affordances of
	// all digital twin TD have the same forms with the hub as endpoint.
	// This results in massive duplication of TD content and increases the TD by a lot.
	// Don't use forms unless you really need it.
	IncludeForms bool `yaml:"includeForms,omitempty"`

	// Enable the MQTT protocol binding, default is false.
	//EnableMQTT bool `yaml:"enableMQTT"`

	// The service discovery instance. The default is the hostname
	DiscoveryInstanceName string `yaml:"instanceName,omitempty"`

	// DirectoryTDPath contains the HTTP path to read the digitwin directory TD
	// Defaults to "/.well-known/wot" as per spec
	// This is published by discovery and served by the http server.
	DirectoryTDPath string `json:"directoryTDPath,omitempty"`

	// Server hostname used in http
	HttpHost string `yaml:"host"`
	// https listening port
	HttpsPort int `yaml:"httpsPort"`

	// Static file server config. Only used when EnableHttpStatic is true
	// HttpStaticBase base path for static file server.
	// Default is /static
	HttpStaticBase string `yaml:"httpStaticBase,omitempty"`
	// HttpStaticDirectory. Storage location of the static file server.
	// Default is {home}/stores/httpstatic
	HttpStaticDirectory string `yaml:"httpStaticDirectory,omitempty"`

	// HiveOT sse subprotocol connection path
	// The full URL is included in discovery record parameters
	// with this URL no forms are needed to connect to the hub
	HiveotSSEPath string `yaml:"hiveotSsePath"`

	// MQTT host interface
	MqttHost string `yaml:"mqttHost"`
	// MQTT tcp port
	MqttTcpPort int `yaml:"mqttTcpPort"`
	// MQTT websocket port
	MqttWssPort int `yaml:"mqttWssPort"`

	// Websocket subprotocol connection path
	// The full URL is included in discovery record parameters
	// with this URL no forms are needed to connect to the hub
	WSSPath string `yaml:"wssPath"`
}

// NewProtocolsConfig creates the default configuration of communication protocols
// This enables https and mdns
func NewProtocolsConfig() ProtocolsConfig {
	hostName, _ := os.Hostname()

	cfg := ProtocolsConfig{
		DirectoryTDPath:       discoserver.DefaultHttpGetDirectoryTDPath,
		EnableHiveotAuth:      true,
		EnableHiveotSSE:       true,
		EnableHttpBasic:       true,
		EnableWSS:             true,
		EnableDiscovery:       true,
		HiveotSSEPath:         hiveotsseserver.DefaultHiveotSsePath,
		HttpStaticBase:        httpbasic.DefaultHttpStaticBase,
		HttpStaticDirectory:   httpbasic.DefaultHttpStaticDirectory,
		IncludeForms:          true, // for interoperability
		DiscoveryInstanceName: hostName,
		HttpHost:              "",
		HttpsPort:             DefaultHttpsPort,
		MqttHost:              hostName,
		MqttTcpPort:           DefaultMqttTcpPort,
		MqttWssPort:           DefaultMqttWssPort,
		WSSPath:               wssserver.DefaultWssPath,
	}
	return cfg
}
