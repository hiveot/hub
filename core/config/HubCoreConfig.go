package config

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/mqttmsgserver"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/utils"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubCoreConfig with core server and services configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	// Run the core using nats or mqtt. (currently only nats)
	Core string `yaml:"core"`

	// The home directory used in init and setup
	HomeDir string `yaml:"homeDir,omitempty"`

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

	Auth authservice.AuthConfig `yaml:"auth"`
}

// Setup creates and loads certificate and key files
// if new is false then re-use existing certificate and key files.
// if new is true then create a whole new empty environment in the home directory
//
//	homeDir is the default data home directory ($HOME)
//	configFile to load or "" to use defaults
//	new to initialize a new environment and delete existing data (careful!)
func (cfg *HubCoreConfig) Setup(homeDir string, configFile string, new bool) error {
	var err error
	cwd, _ := os.Getwd()
	slog.Info("running setup",
		slog.Bool("--new", new),
		slog.String("home", homeDir),
		slog.String("cwd", cwd))

	// 1: Determine directories
	// default to the parent folder of the application binary
	if homeDir == "" {
		homeDir = path.Base(path.Base(os.Args[0]))
	} else if !path.IsAbs(homeDir) {
		homeDir = path.Join(cwd, homeDir)
	}
	f := utils.GetFolders(homeDir, false)
	cfg.HomeDir = f.Home

	// 2: Setup files and folders
	// In a new environment, clear the home directory
	// This is very destructive!
	// do a sanity check on home
	if path.Clean(cfg.HomeDir) == "/etc" {
		panic("Home cannot be /etc")
	} else if path.Clean(cfg.HomeDir) == "/tmp" {
		panic("Home cannot be /tmp. Choose a subdir")
	} else if path.Clean(path.Dir(cfg.HomeDir)) == "/home" {
		panic("application home directory cannot be someone's home directory")
	}
	if _, err2 := os.Stat(cfg.HomeDir); err2 == nil && new {
		slog.Warn("setup new. Removing ",
			"config", f.Config, "certs", f.Certs, "", "stores", f.Stores, "logs", f.Logs)
		_ = os.RemoveAll(f.Config)
		_ = os.RemoveAll(f.Certs)
		_ = os.RemoveAll(f.Stores)
		_ = os.RemoveAll(f.Logs)
	}
	if _, err2 := os.Stat(cfg.HomeDir); err2 != nil {
		err = os.MkdirAll(cfg.HomeDir, 0755)
	}
	if err != nil {
		panic("unable to create home directory: " + cfg.HomeDir)
	}
	cfg.CaCertFile = certs.DefaultCaCertFile
	cfg.CaKeyFile = certs.DefaultCaKeyFile
	cfg.ServerCertFile = "hubCert.pem"
	cfg.ServerKeyFile = "hubKey.pem"

	// 3: Load config file if given
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("unable to load config: %w", err)
		}
		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return fmt.Errorf("unable to parse config: %w", err)
		}
	}

	// 4: Create/Load the CA and server certificates
	cfg.SetupCerts(f.Certs, true)
	cfg.NatsServer.CaCert = cfg.CaCert
	cfg.NatsServer.CaKey = cfg.CaKey
	cfg.NatsServer.ServerTLS = cfg.ServerTLS

	// 5: Setup messaging server config
	err = cfg.NatsServer.Setup(f.Certs, f.Stores, true)

	if cfg.NatsServer.DataDir == "" {
		panic("config is missing server data directory")
	}
	if new {
		_ = os.RemoveAll(cfg.NatsServer.DataDir)
	}
	if _, err2 := os.Stat(cfg.NatsServer.DataDir); err2 != nil {
		slog.Warn("Creating server data directory: " + cfg.NatsServer.DataDir)
		err = os.MkdirAll(cfg.NatsServer.DataDir, 0700)
	}
	if err != nil {
		panic("error creating data directory: " + err.Error())
	}

	// 4: setup authn config
	err = cfg.Auth.Setup(f.Stores)
	if err != nil {
		return err
	}
	return err
}

// SetupCerts load or generate certificates.
// If certificates are preloaded then do nothing.
// If a CA doesn't exist then generate and save a new self-signed cert.
// The server certificates is always regenerated and saved.
// This panics if certs cannot be setup.
func (cfg *HubCoreConfig) SetupCerts(certsDir string, writeChanges bool) {
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
		//
		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365*10)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}
		if writeChanges {

			err = certs.SaveKeysToPEM(cfg.CaKey, caKeyPath)
			if err == nil {
				err = certs.SaveX509CertToPEM(cfg.CaCert, caCertPath)
			}
			if err != nil {
				panic("Unable to save the CA cert or key: " + err.Error())
			}
		}
	}

	// 3: Always create a new server cert and private key
	serverCertPath := cfg.ServerCertFile
	if !path.IsAbs(serverCertPath) {
		serverCertPath = path.Join(certsDir, serverCertPath)
	}
	serverKeyPath := cfg.ServerKeyFile
	if !path.IsAbs(serverKeyPath) {
		serverKeyPath = path.Join(certsDir, serverKeyPath)
	}
	cfg.ServerKey, _ = certs.CreateECDSAKeys()
	hostName, _ := os.Hostname()
	serverID := "nats-" + hostName
	ou := "hiveot"
	names := []string{"localhost", "127.0.0.1", hostName}
	// the server cert is valid for 1 year, after which a restart is needed.
	serverCert, err := certs.CreateServerCert(
		serverID, ou, 365, &cfg.ServerKey.PublicKey, names, cfg.CaCert, cfg.CaKey)
	if err != nil {
		panic("Unable to create a server cert: " + err.Error())
	}
	cfg.ServerTLS = certs.X509CertToTLS(serverCert, cfg.ServerKey)

	if writeChanges {
		err = certs.SaveTLSCertToPEM(cfg.ServerTLS, serverCertPath, serverKeyPath)
	}
}

// NewHubCoreConfig creates a new configuration for the hub server and core services.
// Call Setup to load a config file and update directories.
func NewHubCoreConfig() *HubCoreConfig {
	return &HubCoreConfig{}
}
