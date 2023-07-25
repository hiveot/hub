package config

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubCoreConfig with Hub core configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	// The home directory used in init and setup
	HomeDir string       `json:"homeDir,omitempty"`
	Server  ServerConfig `yaml:"server"`
	Authn   AuthnConfig  `yaml:"authn"`
	Authz   AuthzConfig  `yaml:"authz"`
}

// Setup creates and loads certificate and key files
// if new is false then re-use existing certificate and key files.
// if new is true then create a whole new empty environment in the home directory
func (cfg *HubCoreConfig) Setup(new bool) (
	appAcctKey nkeys.KeyPair,
	serverTLS *tls.Certificate,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey) {
	var err error
	var alwaysNewServerCert = false

	slog.Info("running setup",
		slog.Bool("--new", new), slog.String("home", cfg.HomeDir))

	// 1. in a new environment, clear the home directory
	// this is very destructive
	// do a sanity check on home
	base := path.Base(cfg.HomeDir)
	_ = base
	parent := path.Dir(cfg.HomeDir)
	_ = parent
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

	// 2: handle the CA certificate
	certsDir := path.Dir(cfg.Server.CaCertFile)
	if err2 := os.MkdirAll(certsDir, 0755); err2 != nil && err != os.ErrExist {
		errMsg := fmt.Errorf("unable to create certs directory '%s': %w", certsDir, err.Error())
		panic(errMsg)
	}
	if _, err2 := os.Stat(cfg.Server.CaKeyFile); err2 == nil && !new {
		slog.Info("loading CA certificate and key")
		// load the CA cert and key
		caKey, err = certsclient.LoadKeysFromPEM(cfg.Server.CaKeyFile)
		if err == nil {
			caCert, err = certsclient.LoadX509CertFromPEM(cfg.Server.CaCertFile)
		}
		if err != nil {
			panic("unable to CA certificate: " + err.Error())
		}
	} else {
		slog.Warn("creating new CA certificate and key")
		caCert, caKey, err = certs.CreateCA("hiveot", 365*3)
		if err != nil {
			panic("Unable to create a CA cert: " + err.Error())
		}
		err = certsclient.SaveKeysToPEM(caKey, cfg.Server.CaKeyFile)
		if err == nil {
			err = certsclient.SaveX509CertToPEM(caCert, cfg.Server.CaCertFile)
		}
		// and delete the certs that are based on the key
		_ = os.Remove(cfg.Server.ServerKeyFile)
		_ = os.Remove(cfg.Server.ServerCertFile)
	}
	// 3: handle the Server key
	if _, err2 := os.Stat(cfg.Server.ServerKeyFile); err2 != nil || new || alwaysNewServerCert {
		// create a new server cert private key and cert
		slog.Warn("Creating new server key")
		serverKey := certsclient.CreateECDSAKeys()
		err = certs.SaveKeysToPEM(serverKey, cfg.Server.ServerKeyFile)
		if err != nil {
			panic("Unable to save server key: " + err.Error())
		}
		// with a new key any old cert is useless
		err = os.Remove(cfg.Server.ServerCertFile)
	}
	// 4: handle the Server cert
	if _, err2 := os.Stat(cfg.Server.ServerCertFile); err2 != nil || new {
		slog.Warn("Creating new server cert")
		serverKey, _ := certs.LoadKeysFromPEM(cfg.Server.ServerKeyFile)
		hostName, _ := os.Hostname()
		serverID := "nats-" + hostName
		ou := "hiveot"
		names := []string{"localhost", "127.0.0.1", cfg.Server.Host}
		serverCert, err := certs.CreateServerCert(
			serverID, ou, &serverKey.PublicKey, names, 365, caCert, caKey)
		if err != nil {
			panic("Unable to create a server cert: " + err.Error())
		}
		err = certs.SaveX509CertToPEM(serverCert, cfg.Server.ServerCertFile)
		if err != nil {
			panic("Unable to save server cert")
		}
	}
	// 5: finally load the server TLS
	serverTLS, err = certs.LoadTLSCertFromPEM(cfg.Server.ServerCertFile, cfg.Server.ServerKeyFile)
	if err != nil {
		panic("Unable to load server TLS cert. Is it malformed?: " + err.Error())
	}

	// 6: make sure the server storage dir exists
	if cfg.Server.DataDir == "" {
		panic("config is missing server data directory")
	}
	if new {
		_ = os.RemoveAll(cfg.Server.DataDir)
	}
	if _, err2 := os.Stat(cfg.Server.DataDir); err2 != nil {
		slog.Warn("Creating server data directory: " + cfg.Server.DataDir)
		err = os.MkdirAll(cfg.Server.DataDir, 0700)
	}
	if err != nil {
		panic("error creating data directory: " + err.Error())
	}

	// 7: App account key
	if _, err2 := os.Stat(cfg.Server.AppAccountKeyFile); err2 != nil {
		slog.Warn("Creating application account key file: " + cfg.Server.AppAccountKeyFile)
		appAcctKey, _ = nkeys.CreateAccount()
		appSeed, _ := appAcctKey.Seed()
		err = os.WriteFile(cfg.Server.AppAccountKeyFile, appSeed, 0400)
	} else {
		appSeed, _ := os.ReadFile(cfg.Server.AppAccountKeyFile)
		appAcctKey, err = nkeys.FromSeed(appSeed)
	}
	if err != nil {
		panic("error creating appAcctKey: " + err.Error())
	}

	// 8: authn directories
	if _, err2 := os.Stat(cfg.Authn.CertsDir); err2 != nil {
		err = os.MkdirAll(cfg.Authn.CertsDir, 0755)
	}
	if _, err2 := os.Stat(path.Base(cfg.Authn.PasswordFile)); err2 != nil && err == nil {
		err = os.MkdirAll(path.Base(cfg.Authn.PasswordFile), 0700)
	}
	if err != nil {
		panic("error creating authn directories: " + err.Error())
	}

	// 9: authz directories
	if _, err2 := os.Stat(cfg.Authz.GroupsDir); err2 != nil {
		err = os.MkdirAll(cfg.Authz.GroupsDir, 0700)
	}
	if err != nil {
		panic("error creating authz directory: " + err.Error())
	}
	slog.Info("setup completed successfully")
	return appAcctKey, serverTLS, caCert, caKey
}

// NewHubCoreConfig creates and initalizes a configuration for the hub server and core services.
// This ensures that all fields have valid default values.
//
//	home dir of the application home. Default is the parent of the application bin folder
//	configfile with the name of the config file or "" to not load a config
func NewHubCoreConfig(homeDir string, configFile string) (*HubCoreConfig, error) {
	// default to the parent folder of the application binary
	if homeDir == "" {
		homeDir = path.Base(path.Base(os.Args[0]))
	}
	f := svcconfig.GetFolders(homeDir, false)
	hubCfg := &HubCoreConfig{HomeDir: f.Home}
	// load config file if given
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return hubCfg, fmt.Errorf("unable to load config: %w", err)
		}
		err = yaml.Unmarshal(data, hubCfg)
		if err != nil {
			return hubCfg, fmt.Errorf("unable to parse config: %w", err)
		}
	}
	// ensure that all configuration is valid
	err := hubCfg.Server.InitConfig(f.Certs, f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("server config error: %w", err)
	}
	err = hubCfg.Authn.InitConfig(f.Certs, f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("authn config error: %w", err)
	}
	err = hubCfg.Authz.InitConfig(f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("authz config error: %w", err)
	}

	return hubCfg, nil
}
