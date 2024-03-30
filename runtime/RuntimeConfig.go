package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/protocolbindings/httpsbinding"
)

// RuntimeConfig holds the digital twin runtime and protocol bindings configuration
type RuntimeConfig struct {

	// Enable the GRPC protocol binding, default is false.
	EnableGRPC bool `yaml:"enableGRPC,omitempty"`
	// Enable the HTTPS protocol binding, default is false.
	EnableHTTPS bool `yaml:"enableHTTPS,omitempty"`
	// Enable the MQTT protocol binding, default is false.
	EnableMQTT bool `yaml:"enableMQTT,omitempty"`
	// Enable the NATS protocol binding, default is false.
	EnableNATS bool `yaml:"enableNATS,omitempty"`

	// each protocol binding has its own config section
	HttpsBindingConfig *httpsbinding.HttpsBindingConfig
	//MqttBindingConfig  *MqttBindingConfig
	//NatsBindingConfig  *NatsBindingConfig
	//GrpcBindingConfig  *GrpcBindingConfig

	// Runtime logging
	LogLevel string `yaml:"logLevel,omitempty"` // default: warn
	LogFile  string `yaml:"logFile,omitempty"`  // default: no logfile

	// Runtime data directory for storage of digital twins
	DataDir string `yaml:"dataDir,omitempty"` // default is server default

	// The certs and keys to use by the runtime.
	// These are set directly on startup.
	CaCert    *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey     keys.IHiveKey     `yaml:"-"` // preset, load, or error
	ServerKey keys.IHiveKey     `yaml:"-"` // generated, loaded  (used as signing key)
	ServerTLS *tls.Certificate  `yaml:"-"` // generated

}

// NewRuntimeConfig returns a new runtime configuration instance with default values
// This can be used out of the box or be loaded from a yaml configuration file.
//
// The CA and Server certificate and keys must be set after creation.
func NewRuntimeConfig() *RuntimeConfig {
	config := &RuntimeConfig{
		EnableHTTPS:        false,
		EnableMQTT:         false,
		EnableNATS:         false,
		EnableGRPC:         false,
		HttpsBindingConfig: httpsbinding.NewHttpsBindingConfig(),
		LogLevel:           "warning", // error, warning, info, debug
		LogFile:            "",        // no logfile
	}
	return config
}
