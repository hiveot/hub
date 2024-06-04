package mqtttransport

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient/mqttclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/net"
	"log/slog"
	"path"
)

// MqttServerConfig holds the mqtt broker configuration
type MqttServerConfig struct {
	// Host is the server address, default is outbound IP address
	Host string `yaml:"host,omitempty"`
	// Port is the server TLS port, default is 8883
	Port int `yaml:"port,omitempty"`
	// WSPort is the server Websocket port, default is 8884
	WSPort int `yaml:"wsPort,omitempty"`

	LogLevel string `yaml:"logLevel,omitempty"` // default: warn
	LogFile  string `yaml:"logFile,omitempty"`  // default: no logfile

	DataDir string `yaml:"dataDir,omitempty"` // default is server default

	// Disable running the embedded messaging server. Default False
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`

	// the in-proc UDS name to use. Default is "@/MqttInMemUDSProd" (see MqttHubTransport)
	InMemUDSName string `yaml:"inMemUDSName,omitempty"`

	// The certs and keys are set directly
	CaCert    *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey     keys.IHiveKey     `yaml:"-"` // preset, load, or error
	ServerKey keys.IHiveKey     `yaml:"-"` // generated, loaded  (used as signing key)
	ServerTLS *tls.Certificate  `yaml:"-"` // generated

	// Core Service credentials for use by in-proc connection
	//CoreServiceKP  *ecdsa.PrivateKey `yaml:"-"` // generated
	//CoreServicePub string            `yaml:"-"` // generated

	// The following options are JWT specific
}

// Setup the mqtt server config.
// This applies sensible defaults to config.
//
// Any existing values that are previously set remain unchanged.
// Missing values are created.
// Certs and keys are loaded if not provided.
//
// Set 'writeChanges' to persist generated server cert, operator and account keys
//
//	keysDir is the default key location
//	storesDir is the data storage root (default $HOME/stores)
//	writeChanges writes generated account key to the keysDir
func (cfg *MqttServerConfig) Setup(keysDir, storesDir string, writeChanges bool) (err error) {

	// Step 1: Apply defaults parameters
	if cfg.Host == "" {
		outboundIP := net.GetOutboundIP("")
		cfg.Host = outboundIP.String()
	}
	if cfg.Port == 0 {
		cfg.Port = 8883
	}
	if cfg.WSPort == 0 {
		cfg.WSPort = 8884
	}
	if cfg.DataDir == "" {
		cfg.DataDir = path.Join(storesDir, "mqttserver")
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}
	if cfg.InMemUDSName == "" {
		cfg.InMemUDSName = mqttclient.MqttInMemUDSProd
	}

	// Step 2: generate missing certificates
	// These are typically set directly before running setup so this is intended
	// for testing.
	if cfg.CaCert == nil || cfg.CaKey == nil {
		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365)
	}
	if cfg.ServerKey == nil {
		slog.Warn("Creating server key")
		cfg.ServerKey = keys.NewKey(keys.KeyTypeECDSA)
	}
	if cfg.ServerTLS == nil && cfg.CaKey != nil {
		names := []string{cfg.Host}
		serverX509, err := certs.CreateServerCert(
			"hiveot", "server",
			365, // validity matches the CA
			cfg.ServerKey,
			names, cfg.CaCert, cfg.CaKey)
		if err != nil {
			slog.Error("unable to generate server cert. Not using TLS.", "err", err)
		} else {
			cfg.ServerTLS = certs.X509CertToTLS(serverX509, cfg.ServerKey)
		}
	}

	// Step 4: generate admin keys and token
	// core service keys are always regenerated and not saved
	//if cfg.CoreServiceKP == nil {
	//	cfg.CoreServiceKP, cfg.CoreServicePub = certs.CreateECDSAKeys()
	//}

	return nil
}
