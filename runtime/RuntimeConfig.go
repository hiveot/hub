package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/plugin"
	service2 "github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authz/service"
	"github.com/hiveot/hub/transports/servers"
	"github.com/hiveot/hub/transports/tputils/net"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
)

const DefaultServerCertFile = "hubCert.pem"

// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
const DefaultServerKeyFile = "hubKey.pem"

// RuntimeConfig holds the digital twin runtime and protocol bindings configuration:
// * determines the config, data and certificate storage paths
// * load or generate the CA and server keys and certificates
type RuntimeConfig struct {

	// middleware and services config. These all work out of the box with their defaults.
	Authn          service2.AuthnConfig    `yaml:"authn"`
	Authz          service.AuthzConfig     `yaml:"authz"`
	ProtocolConfig servers.ProtocolsConfig `yaml:"protocols"`

	// Runtime logging
	LogLevel string `yaml:"logLevel,omitempty"` // default: warn
	//LogFile  string `yaml:"logFile,omitempty"`  // default: no logfile

	// RequestLog with logging of request and response messages
	// file is <requests>-<yyyy-mm-dd>.log
	RequestLog string `yaml:"requestLog,omitempty"` // default: no log file
	// NotifLog filename for logging of notification messages
	NotifLog string `yaml:"notifLog,omitempty"`
	// RuntimeLog filename for other runtime messages
	RuntimeLog string `yaml:"runtimeLog,omitempty"`

	//RequestLogStdout bool   `yaml:"requestLogStdout"`     // fork to stdout
	LogfileInJson bool `yaml:"logfileInJson"` // logfile in json format

	// Runtime data directory for storage of digital twins
	DataDir string `yaml:"dataDir,omitempty"` // default is server default

	// certificate file names or full path
	CaCertFile string `yaml:"caCertFile"` // default: caCert.pem
	// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
	CaKeyFile      string `yaml:"caKeyFile"`      // default: caKey.pem
	ServerCertFile string `yaml:"serverCertFile"` // default: hubCert.pem
	// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
	ServerKeyFile string `yaml:"serverKeyFile"` // default: hubKey.pem
	// The certs and keys to use by the runtime.
	// These are set directly on startup.
	CaCert     *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey      keys.IHiveKey     `yaml:"-"` // preset, load, or error
	ServerKey  keys.IHiveKey     `yaml:"-"` // generated, loaded  (used as signing key)
	ServerCert *tls.Certificate  `yaml:"-"` // generated

}

// SetupCerts load or generate certificates.
// If certificates are preloaded then do nothing.
// If a CA doesn't exist then generate and save a new self-signed cert valid for localhost,127.0.0.1 and outbound IP
// The server certificates is always regenerated and saved.
// This panics if certs cannot be setup.
func (cfg *RuntimeConfig) setupCerts(env *plugin.AppEnvironment) {
	var err error
	certsDir := env.CertsDir

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
	} else {
		// save the caCert and its key
		err = certs.SaveX509CertToPEM(cfg.CaCert, caCertPath)
		if err == nil && cfg.CaKey != nil {
			err = cfg.CaKey.ExportPrivateToFile(caKeyPath)
		}
	}
	// only load the ca key if the cert was loaded
	if cfg.CaCert != nil && cfg.CaKey == nil {
		cfg.CaKey, err = keys.NewKeyFromFile(caKeyPath)
	}

	// 2: if no CA exists, create a self-signed certificate
	if err != nil || cfg.CaCert == nil || cfg.CaKey == nil {
		slog.Warn("creating a self-signed CA certificate and key", "caCertPath", caCertPath)

		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365*10)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}

		err = cfg.CaKey.ExportPrivateToFile(caKeyPath)
		if err == nil {
			err = certs.SaveX509CertToPEM(cfg.CaCert, caCertPath)
		}
		if err != nil {
			panic("Unable to save the CA cert or key: " + err.Error())
		}
	}

	// 3: Load or create a new server private key if it doesn't exist
	// As this key is used to sign auth tokens, save it after creation
	serverKeyPath := cfg.ServerKeyFile
	if !path.IsAbs(serverKeyPath) {
		serverKeyPath = path.Join(certsDir, serverKeyPath)
	}

	// load the server key if available
	newServerKey := false
	if cfg.ServerKey == nil {
		slog.Info("Loading server key", "serverKeyPath", serverKeyPath)
		cfg.ServerKey, err = keys.NewKeyFromFile(serverKeyPath)
	} else {
		slog.Info("Using provided server key")
		err = nil
	}
	if err != nil || cfg.ServerKey == nil {
		newServerKey = true
		slog.Warn("Creating server key")
		cfg.ServerKey = keys.NewKey(cfg.CaKey.KeyType()) // use same key type as CA
		err = cfg.ServerKey.ExportPrivateToFile(serverKeyPath)
		if err != nil {
			slog.Error("Unable to save the server key", "err", err)
		}
	}
	// load or create a new server cert
	serverCertPath := cfg.ServerCertFile
	if !path.IsAbs(serverCertPath) {
		serverCertPath = path.Join(certsDir, serverCertPath)
	}
	hostName, _ := os.Hostname()
	if cfg.ServerCert == nil || !newServerKey {
		cfg.ServerCert, err = certs.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
		if err == nil {
			slog.Info("loaded Server certificate and key")
		}
	}

	if cfg.ServerCert == nil {
		serverID := "dtr-" + hostName
		ou := "hiveot"
		outboundIP := net.GetOutboundIP("")
		names := []string{"localhost", "127.0.0.1", hostName, outboundIP.String()}

		// regenerate a new server cert, valid for 1 year
		serverCert, err := certs.CreateServerCert(
			serverID, ou, 365, cfg.ServerKey, names, cfg.CaCert, cfg.CaKey)
		if err != nil {
			panic("Unable to create a server cert: " + err.Error())
		}
		cfg.ServerCert = certs.X509CertToTLS(serverCert, cfg.ServerKey)

		slog.Info("Writing server cert", "serverCertPath", serverCertPath)
		err = certs.SaveX509CertToPEM(serverCert, serverCertPath)
		if err != nil {
			slog.Error("writing server cert failed: ", "err", err)
		}
	}
}

// setupDirectories creates missing directories
// parameter new deletes existing data directories first. Careful!
func (cfg *RuntimeConfig) setupDirectories(env *plugin.AppEnvironment) error {

	if path.Clean(env.HomeDir) == "/etc" {
		return fmt.Errorf("home cannot be /etc")
	} else if path.Clean(env.HomeDir) == "/tmp" {
		return fmt.Errorf("home cannot be /tmp. Choose a subdir")
	} else if path.Clean(path.Dir(env.HomeDir)) == "/home" {
		return fmt.Errorf("application home directory cannot be someone's home directory")
	}
	//if _, err2 := os.Stat(env.HomeDir); err2 == nil && new {
	//	println("Setup new. Removing certs, stores and logs directories")
	//	// keep old config as there is no way to re-install defaults
	//	//_ = os.RemoveAll(env.ConfigDir)
	//	_ = os.RemoveAll(env.CertsDir)
	//	_ = os.RemoveAll(env.StoresDir)
	//	_ = os.RemoveAll(env.LogsDir)
	//}

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

// Setup the runtime environment along with certificate and key files.
// This:
// 1. Load the runtime config file if any exists
// 2. Creates missing directories
// 3. Create missing certificates, including a self-signed CA.
// 5. Create auth keys for certs and launcher services
// 6. Create a default launcher config if none exists
//
//	env holds the application directory environment
func (cfg *RuntimeConfig) Setup(env *plugin.AppEnvironment) error {
	var err error
	//logging.SetLogging(cfg.LogLevel, cfg.LogFile)
	slog.Info("Digital Twin Runtime setup",
		slog.String("home", env.HomeDir),
		slog.String("config", env.ConfigFile),
	)

	// 0: Load config file if given
	if _, err = os.Stat(env.ConfigFile); err == nil {
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
	err = cfg.setupDirectories(env)
	if err != nil {
		return err
	}
	// 3: Setup certificates
	cfg.setupCerts(env)

	// 4: setup authn config
	cfg.Authn.Setup(env.CertsDir, env.StoresDir)

	// 5: setup authz config
	cfg.Authz.Setup(env.StoresDir)

	// 4. configure protocol bindings
	return err
}

// NewRuntimeConfig returns a new runtime configuration instance with default values
// This can be used out of the box or be loaded from a yaml configuration file.
//
// The CA and Server certificate and keys must be set after creation.
func NewRuntimeConfig() *RuntimeConfig {
	cfg := &RuntimeConfig{
		Authn:          service2.NewAuthnConfig(),
		Authz:          service.NewAuthzConfig(),
		ProtocolConfig: servers.NewProtocolsConfig(),
		LogLevel:       "info", // error, warning, info, debug
		NotifLog:       "",     // no logfile
		RequestLog:     "",     // no request logfile
		RuntimeLog:     "",     // no logfile

		CaCertFile:     certs.DefaultCaCertFile,
		CaKeyFile:      certs.DefaultCaKeyFile,
		ServerCertFile: DefaultServerCertFile,
		ServerKeyFile:  DefaultServerKeyFile,
	}
	return cfg
}
