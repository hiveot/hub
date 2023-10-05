package config

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
)

// HubCoreConfig with core server and services configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	Env utils.AppEnvironment

	// certificate file names or full path
	CaCertFile     string            `yaml:"caCertFile"`     // default: caCert.pem
	CaKeyFile      string            `yaml:"caKeyFile"`      // default: caKey.pem
	ServerCertFile string            `yaml:"serverCertFile"` // default: hubCert.pem
	ServerKeyFile  string            `yaml:"serverKeyFile"`  // default: kubKey.pem
	CaCert         *x509.Certificate `yaml:"-"`              // preset, load, or error
	CaKey          *ecdsa.PrivateKey `yaml:"-"`              // preset, load, or error
	ServerTLS      *tls.Certificate  `yaml:"-"`              // preset, load, or generate
	ServerKey      *ecdsa.PrivateKey `yaml:"-"`

	// use either nats or mqtt.
	NatsServer natsmsgserver.NatsServerConfig `yaml:"natsserver"`
	MqttServer mqttmsgserver.MqttServerConfig `yaml:"mqttserver"`

	// auth service config
	Auth auth.AuthConfig `yaml:"auth"`

	// enable mDNS discovery
	EnableMDNS bool `yaml:"enableMDNS"`
}

// Setup creates and loads certificate and key files
// The core selection determines the type of auth keys to generated for core service and admin.
// if new is false then re-use existing certificate and key files.
// if new is true then create a whole new empty environment in the home directory
//
//	homeDir is the default data home directory ($HOME)
//	configFile to load or "" to use defaults
//	core to setup, "nats" or "mqtt"
//	new to initialize a new environment and delete existing data (careful!)
func (cfg *HubCoreConfig) Setup(env utils.AppEnvironment, new bool) error {
	var err error
	slog.Info("running setup",
		slog.Bool("--new", new),
		slog.String("home", env.HomeDir),
	)

	cfg.Env = env

	// 1: Setup files and folders
	// In a new environment, clear the home directory.
	// This is very destructive!
	// Do a sanity check on home first
	if path.Clean(env.HomeDir) == "/etc" {
		panic("Home cannot be /etc")
	} else if path.Clean(env.HomeDir) == "/tmp" {
		panic("Home cannot be /tmp. Choose a subdir")
	} else if path.Clean(path.Dir(env.HomeDir)) == "/home" {
		panic("application home directory cannot be someone's home directory")
	}
	if _, err2 := os.Stat(env.HomeDir); err2 == nil && new {
		slog.Warn("setup new. Removing ",
			slog.String("config", env.ConfigDir),
			slog.String("certs", env.CertsDir),
			slog.String("stores", env.StoresDir),
			slog.String("logs", env.LogsDir))
		_ = os.RemoveAll(env.ConfigDir)
		_ = os.RemoveAll(env.CertsDir)
		_ = os.RemoveAll(env.StoresDir)
		_ = os.RemoveAll(env.LogsDir)
	}
	if _, err2 := os.Stat(env.HomeDir); err2 != nil {
		err = os.MkdirAll(env.HomeDir, 0755)
	}
	if err != nil {
		panic("unable to create home directory: " + env.HomeDir)
	}
	cfg.CaCertFile = certs.DefaultCaCertFile
	cfg.CaKeyFile = certs.DefaultCaKeyFile
	cfg.ServerCertFile = "hubCert.pem"
	cfg.ServerKeyFile = "hubKey.pem"

	// 2: Load config file if given
	if _, err := os.Stat(env.ConfigFile); err == nil {
		data, err := os.ReadFile(env.ConfigFile)
		if err != nil {
			return fmt.Errorf("unable to load config: %w", err)
		}
		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return fmt.Errorf("unable to parse config: %w", err)
		}
	}

	// 3: Create/Load the CA and server certificates
	cfg.SetupCerts(env.CertsDir)

	// 4: Setup nats config
	if env.ServerCore == "nats" {
		err = cfg.SetupNatsCore(env)
	} else {
		// 6: Setup mqtt config
		cfg.MqttServer.CaCert = cfg.CaCert
		cfg.MqttServer.CaKey = cfg.CaKey
		cfg.MqttServer.ServerTLS = cfg.ServerTLS
		err = cfg.MqttServer.Setup(env.CertsDir, env.StoresDir, true)
	}
	// 5: setup authn config
	err = cfg.Auth.Setup(env.CertsDir, env.StoresDir)
	if err != nil {
		return err
	}
	return err
}

// SetupCerts load or generate certificates.
// If certificates are preloaded then do nothing.
// If a CA doesn't exist then generate and save a new self-signed cert valid for localhost,127.0.0.1 and outbound IP
// The server certificates is always regenerated and saved.
// This panics if certs cannot be setup.
func (cfg *HubCoreConfig) SetupCerts(certsDir string) {
	var err error

	// setup files and folders
	if _, err = os.Stat(certsDir); err != nil {
		if err2 := os.MkdirAll(certsDir, 0755); err2 != nil && errors.Is(err, os.ErrExist) {
			errMsg := fmt.Errorf("unable to create certs directory '%s': %w", certsDir, err)
			panic(errMsg)
		}
	}

	caCertPath := cfg.CaCertFile
	if !path.IsAbs(caCertPath) {
		caCertPath = path.Join(certsDir, caCertPath)
	}
	caKeyPath := cfg.CaKeyFile
	if !path.IsAbs(caKeyPath) {
		caKeyPath = path.Join(certsDir, caKeyPath)
	}
	// 1: load the CA if available
	if cfg.CaCert == nil {
		slog.Info("loading CA certificate and key")
		cfg.CaCert, err = certs.LoadX509CertFromPEM(caCertPath)
	}
	// only load the ca key if the cert was loaded
	if cfg.CaCert != nil && cfg.CaKey == nil {
		cfg.CaKey, err = certs.LoadKeysFromPEM(caKeyPath)
	}

	// 2: if no CA exists, create it
	if cfg.CaCert == nil || cfg.CaKey == nil {
		slog.Warn("creating a self-signed CA certificate and key", "caCertPath", caCertPath)

		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365*10)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}

		err = certs.SaveKeysToPEM(cfg.CaKey, caKeyPath)
		if err == nil {
			err = certs.SaveX509CertToPEM(cfg.CaCert, caCertPath)
		}
		if err != nil {
			panic("Unable to save the CA cert or key: " + err.Error())
		}
	}

	// 3: Create a new server private key if it doesn't exist
	serverKeyPath := cfg.ServerKeyFile
	if !path.IsAbs(serverKeyPath) {
		serverKeyPath = path.Join(certsDir, serverKeyPath)
	}
	// load the server key if available
	if cfg.ServerKey == nil {
		cfg.ServerKey, _ = certs.LoadKeysFromPEM(serverKeyPath)
	}
	if cfg.ServerKey == nil {
		cfg.ServerKey, _ = certs.CreateECDSAKeys()
	}
	// create a new server cert
	serverCertPath := cfg.ServerCertFile
	if !path.IsAbs(serverCertPath) {
		serverCertPath = path.Join(certsDir, serverCertPath)
	}
	hostName, _ := os.Hostname()
	serverID := "nats-" + hostName
	ou := "hiveot"
	outboundIP := utils.GetOutboundIP("")
	names := []string{"localhost", "127.0.0.1", hostName, outboundIP.String()}

	// regenerate a new server cert, valid for 1 year
	serverCert, err := certs.CreateServerCert(
		serverID, ou, 365, &cfg.ServerKey.PublicKey, names, cfg.CaCert, cfg.CaKey)
	if err != nil {
		panic("Unable to create a server cert: " + err.Error())
	}
	cfg.ServerTLS = certs.X509CertToTLS(serverCert, cfg.ServerKey)

	err = certs.SaveTLSCertToPEM(cfg.ServerTLS, serverCertPath, serverKeyPath)
}

// SetupNatsCore load or generate nats service and admin keys.
func (cfg *HubCoreConfig) SetupNatsCore(f utils.AppEnvironment) error {
	var err error
	slog.Warn("env", "CertsDir", f.CertsDir,
		"HomeDir", f.HomeDir)
	cfg.NatsServer.CaCert = cfg.CaCert
	cfg.NatsServer.CaKey = cfg.CaKey
	cfg.NatsServer.ServerTLS = cfg.ServerTLS
	err = cfg.NatsServer.Setup(f.CertsDir, f.StoresDir, true)

	if cfg.NatsServer.DataDir == "" {
		panic("config is missing server data directory")
	}
	if _, err2 := os.Stat(cfg.NatsServer.DataDir); err2 != nil {
		slog.Warn("Creating server data directory: " + cfg.NatsServer.DataDir)
		err = os.MkdirAll(cfg.NatsServer.DataDir, 0700)
	}
	if err != nil {
		panic("error creating data directory: " + err.Error())
	}
	return err
}

// NewHubCoreConfig creates a new configuration for the hub server and core services.
// Call Setup to load a config file and update directories.
func NewHubCoreConfig() *HubCoreConfig {
	return &HubCoreConfig{
		EnableMDNS: true,
	}
}
