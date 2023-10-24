package config

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/core/auth/config"
	"github.com/hiveot/hub/core/msgserver/mqttmsgserver"
	"github.com/hiveot/hub/core/msgserver/natsmsgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
)

const HubCoreConfigFileName = "hub.yaml"
const DefaultServerCertFile = "hubCert.pem"
const DefaultServerKeyFile = "hubKey.pem"

// HubCoreConfig with core server, auth, cert and launcher configuration
// Used for launching the core.
// Use NewHubCoreConfig to create a default config
// FIXME: this is temporary, each service must handle their own config yaml
type HubCoreConfig struct {
	Env utils.AppEnvironment

	// certificate file names or full path

	CaCertFile     string            `yaml:"caCertFile"`     // default: caCert.pem
	CaKeyFile      string            `yaml:"caKeyFile"`      // default: caKey.pem
	ServerCertFile string            `yaml:"serverCertFile"` // default: hubCert.pem
	ServerKeyFile  string            `yaml:"serverKeyFile"`  // default: hubKey.pem
	CaCert         *x509.Certificate `yaml:"-"`              // preset, load, or error
	CaKey          *ecdsa.PrivateKey `yaml:"-"`              // preset, load, or error
	ServerTLS      *tls.Certificate  `yaml:"-"`              // preset, load, or generate
	ServerKey      *ecdsa.PrivateKey `yaml:"-"`

	// use either nats or mqtt.
	NatsServer natsmsgserver.NatsServerConfig `yaml:"natsserver"`
	MqttServer mqttmsgserver.MqttServerConfig `yaml:"mqttserver"`

	// auth service config
	Auth config.AuthConfig `yaml:"auth"`

	// enable mDNS discovery
	EnableMDNS bool `yaml:"enableMDNS"`
}

// Setup ensures the hub core configuration exists along with certificate and key files.
// This:
// 1. If 'new' is true then delete existing config, certs, logs and storage.
// 2. Creates missing directories
// 3. Create missing certificates, including a self-signed CA.
// 4. Setup the message server config, nats or mqtt
// 5. Create auth keys for certs and launcher services
// 6. Create a default launcher config if none exists
//
//	env holds the application directory environment
//	core holds the core to setup, "nats" or "mqtt" (default)
//	new to initialize a new environment and delete existing data (careful!)
func (cfg *HubCoreConfig) Setup(env *utils.AppEnvironment, core string, new bool) error {
	var err error
	slog.Info("running setup",
		slog.Bool("--new", new),
		slog.String("home", env.HomeDir),
		slog.String("core", core),
	)

	cfg.Env = *env
	//cfg.Env.Core = core

	println("CORE=" + core)

	// 0: Load config file if given
	if _, err := os.Stat(cfg.Env.ConfigFile); err == nil {
		data, err := os.ReadFile(env.ConfigFile)
		if err != nil {
			return fmt.Errorf("unable to load config: %w", err)
		}
		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return fmt.Errorf("unable to parse config: %w", err)
		}
	}

	// 2: Setup directories
	err = cfg.setupDirectories(new)
	if err != nil {
		return err
	}

	// 3: Setup certificates
	cfg.CaCertFile = certs.DefaultCaCertFile
	cfg.CaKeyFile = certs.DefaultCaKeyFile
	cfg.ServerCertFile = DefaultServerCertFile
	cfg.ServerKeyFile = DefaultServerKeyFile
	cfg.setupCerts()
	// pass it on to the message server
	cfg.MqttServer.CaCert = cfg.CaCert
	cfg.MqttServer.CaKey = cfg.CaKey
	cfg.MqttServer.ServerKey = cfg.ServerKey
	cfg.MqttServer.ServerTLS = cfg.ServerTLS
	cfg.NatsServer.CaCert = cfg.CaCert
	cfg.NatsServer.CaKey = cfg.CaKey
	cfg.NatsServer.ServerTLS = cfg.ServerTLS

	// 4: Setup message server config
	if core == "nats" {
		err = cfg.setupNatsCore()
	} else {
		err = cfg.setupMqttCore()
	}
	// 5: setup authn config
	err = cfg.Auth.Setup(cfg.Env.CertsDir, cfg.Env.StoresDir)
	if err != nil {
		return err
	}

	// 6: setup launcher config
	//err = cfg.Launcher.Setup(cfg.Env.CertsDir)
	return err
}

// SetupCerts load or generate certificates.
// If certificates are preloaded then do nothing.
// If a CA doesn't exist then generate and save a new self-signed cert valid for localhost,127.0.0.1 and outbound IP
// The server certificates is always regenerated and saved.
// This panics if certs cannot be setup.
func (cfg *HubCoreConfig) setupCerts() {
	var err error
	certsDir := cfg.Env.CertsDir

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

	// 3: Load or create a new server private key if it doesn't exist
	// As this key is used to sign tokens, save it after creation
	serverKeyPath := cfg.ServerKeyFile
	if !path.IsAbs(serverKeyPath) {
		serverKeyPath = path.Join(certsDir, serverKeyPath)
	}
	// load the server key if available
	if cfg.ServerKey == nil {
		slog.Warn("Loading server key", "serverKeyPath", serverKeyPath)
		cfg.ServerKey, _ = certs.LoadKeysFromPEM(serverKeyPath)
	} else {
		slog.Warn("Using provided server key")
	}
	if cfg.ServerKey == nil {
		slog.Warn("Creating server key")
		cfg.ServerKey, _ = certs.CreateECDSAKeys()
		err = certs.SaveKeysToPEM(cfg.ServerKey, serverKeyPath)
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

	slog.Warn("Writing server cert", "serverCertPath", serverCertPath)
	err = certs.SaveX509CertToPEM(serverCert, serverCertPath)
	if err != nil {
		slog.Error("writing server cert failed: ", "err", err)
	}
	//err = certs.SaveTLSCertToPEM(cfg.ServerTLS, serverCertPath, serverKeyPath)
}

// setupDirectories creates missing directories
// parameter new deletes existing data directories first. Careful!
func (cfg *HubCoreConfig) setupDirectories(new bool) error {
	// In a new environment, clear the home directory.
	// This is very destructive!
	// Do a sanity check on home first
	env := cfg.Env
	if path.Clean(env.HomeDir) == "/etc" {
		return fmt.Errorf("home cannot be /etc")
	} else if path.Clean(env.HomeDir) == "/tmp" {
		return fmt.Errorf("home cannot be /tmp. Choose a subdir")
	} else if path.Clean(path.Dir(env.HomeDir)) == "/home" {
		return fmt.Errorf("application home directory cannot be someone's home directory")
	}
	if _, err2 := os.Stat(env.HomeDir); err2 == nil && new {
		println("Setup new. Removing certs, stores and logs directories")
		// keep old config as there is no way to re-install defaults
		//_ = os.RemoveAll(env.ConfigDir)
		_ = os.RemoveAll(env.CertsDir)
		_ = os.RemoveAll(env.StoresDir)
		_ = os.RemoveAll(env.LogsDir)
	}

	if _, err2 := os.Stat(env.HomeDir); err2 != nil {
		err := os.MkdirAll(env.HomeDir, 0755)
		if err != nil {
			err = fmt.Errorf("unable to create home directory '%s': %w", env.HomeDir, err)
			return err
		}
	}
	// 2. ensure the directories exist
	if _, err2 := os.Stat(env.HomeDir); err2 != nil {
		_ = os.MkdirAll(env.HomeDir, 0755)
	}
	if _, err2 := os.Stat(env.BinDir); err2 != nil {
		_ = os.MkdirAll(env.BinDir, 0755)
	}
	if _, err2 := os.Stat(env.PluginsDir); err2 != nil {
		_ = os.MkdirAll(env.PluginsDir, 0755)
	}
	if _, err2 := os.Stat(env.CertsDir); err2 != nil {
		_ = os.MkdirAll(env.CertsDir, 0755)
	}
	if _, err2 := os.Stat(env.ConfigDir); err2 != nil {
		_ = os.MkdirAll(env.ConfigDir, 0755)
	}
	if _, err2 := os.Stat(env.LogsDir); err2 != nil {
		_ = os.MkdirAll(env.LogsDir, 0755)
	}
	if _, err2 := os.Stat(env.StoresDir); err2 != nil {
		_ = os.MkdirAll(env.StoresDir, 0755)
	}
	return nil
}

// setupNatsCore load or generate nats service and admin keys.
func (cfg *HubCoreConfig) setupMqttCore() error {
	var err error
	slog.Warn("setup mqtt core", "CertsDir", cfg.Env.CertsDir,
		"HomeDir", cfg.Env.HomeDir)
	// 6: Setup mqtt config
	cfg.MqttServer.CaCert = cfg.CaCert
	cfg.MqttServer.CaKey = cfg.CaKey
	cfg.MqttServer.ServerTLS = cfg.ServerTLS
	err = cfg.MqttServer.Setup(cfg.Env.CertsDir, cfg.Env.StoresDir, true)
	return err
}

// setupNatsCore load or generate nats service and admin keys.
func (cfg *HubCoreConfig) setupNatsCore() error {
	var err error
	slog.Warn("setup nats core", "CertsDir", cfg.Env.CertsDir,
		"HomeDir", cfg.Env.HomeDir)
	cfg.NatsServer.CaCert = cfg.CaCert
	cfg.NatsServer.CaKey = cfg.CaKey
	cfg.NatsServer.ServerTLS = cfg.ServerTLS
	err = cfg.NatsServer.Setup(cfg.Env.CertsDir, cfg.Env.StoresDir, true)

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

// Load the core config from hub.yaml
//func (cfg *HubCoreConfig) Load() error {
//	configFile := path.Join(cfg.Env.ConfigDir, HubCoreConfigFileName)
//	data, err := os.ReadFile(configFile)
//	if err != nil {
//		return err
//	}
//	err = yaml.Unmarshal(data, cfg)
//	return err
//}
//
//// Save the core config to hub.yaml
//func (cfg *HubCoreConfig) Save() error {
//	configFile := path.Join(cfg.Env.ConfigDir, HubCoreConfigFileName)
//	data, err := yaml.Marshal(cfg)
//	if err != nil {
//		return err
//	}
//	err = os.WriteFile(configFile, data, 0644)
//	return err
//}

// NewHubCoreConfig creates a new configuration for the hub server and core services.
// Call Setup to load a config file and update directories.
func NewHubCoreConfig() *HubCoreConfig {
	return &HubCoreConfig{
		EnableMDNS: true,
	}
}
