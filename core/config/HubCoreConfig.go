package config

import (
	"fmt"
	"github.com/hiveot/hub/core/authn/authnservice"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/svcconfig"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubCoreConfig with core server and services configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	// The home directory used in init and setup
	HomeDir string `yaml:"homeDir,omitempty"`

	// TODO: certs at this level
	//CaCertFile     string `yaml:"caCertFile"`     // default: caCert.pem
	//CaKeyFile      string `yaml:"caKeyFile"`      // default: caKey.pem
	//ServerCertFile string `yaml:"serverCertFile"` // default: hubCert.pem
	//ServerKeyFile  string `yaml:"serverKeyFile"`  // default: kubKey.pem

	Core string `yaml:"core"` // nats or mqtt

	NatsServer nkeyserver.NatsServerConfig `yaml:"natsserver"`
	//MqttServer  mqttserver.MqttServerConfig `yaml:"mqttserver"`
	Authn authnservice.AuthnConfig `yaml:"authn"`
	Authz authzservice.AuthzConfig `yaml:"authz"`
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

	slog.Info("running setup",
		slog.Bool("--new", new), slog.String("home", cfg.HomeDir))

	// 1: Determine directories
	// default to the parent folder of the application binary
	if homeDir == "" {
		homeDir = path.Base(path.Base(os.Args[0]))
	}
	f := svcconfig.GetFolders(homeDir, false)
	cfg.HomeDir = f.Home

	// 2: Setup the home directory
	// In a new environment, clear the home directory
	// This is very destructive!
	// do a sanity check on home
	if cfg.HomeDir == "" || !path.IsAbs(cfg.HomeDir) {
		panic("home directory is not an absolute path")
	} else if path.Base(cfg.HomeDir) == "etc" {
		panic("Home cannot be /etc")
	} else if path.Base(cfg.HomeDir) == "tmp" {
		panic("Home cannot be /tmp. Choose a subdir")
	} else if path.Base(path.Dir(cfg.HomeDir)) == "home" {
		panic("application home directory cannot be someone's home directory")
	}
	if _, err2 := os.Stat(cfg.HomeDir); err2 == nil && new {
		_ = os.RemoveAll(cfg.HomeDir)
	}
	if _, err2 := os.Stat(cfg.HomeDir); err2 != nil {
		err = os.MkdirAll(cfg.HomeDir, 0755)
	}
	if err != nil {
		panic("unable to create home directory: " + cfg.HomeDir)
	}

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

	// 4: Create/Load the certificates
	serverTLS, caCert, caKey := certs.SetupCerts(
		f.Certs, cfg.NatsServer.CaCertFile, cfg.NatsServer.CaKeyFile)
	cfg.NatsServer.CaCert = caCert
	cfg.NatsServer.CaKey = caKey
	cfg.NatsServer.ServerCert = serverTLS

	// 5: Setup messaging server.
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
	err = cfg.Authn.Setup(f.Stores)
	if err != nil {
		return err
	}
	err = cfg.Authz.Setup(f.Stores)
	if err != nil {
		return err
	}
	return err
}

// NewHubCoreConfig creates a configuration for the hub server and core services.
// Call Setup to ensure that all fields have valid default values.
func NewHubCoreConfig() *HubCoreConfig {
	return &HubCoreConfig{}
}
