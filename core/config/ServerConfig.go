package config

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/nkeys"
	"os"
	"path"
)

// ServerConfig configuration of the messaging server
type ServerConfig struct {
	// Host is the server Hostname must match the name in the server certificate
	Host string `yaml:"host"`
	// Port, default is 4222
	Port int `yaml:"port"`
	//
	WSPort         int    `yaml:"WSPort"`
	ServerCertFile string `yaml:"serverCertFile"`
	ServerKeyFile  string `yaml:"serverKeyFile"`
	StoresDir      string `yaml:"storesDir"`
	CaCertFile     string `yaml:"caCertFile"`
	// application account name and key for issued tokens
	AppAccountName    string `yaml:"appAccountName"`
	AppAccountKeyFile string `yaml:"appAccountKeyFile"`

	// after LoadConfig
	AppAccountKey nkeys.KeyPair
	CaCert        *x509.Certificate
	ServerCert    *tls.Certificate
}

// LoadConfig loads the files used in the configuration
// If no account key file is given or the account key does exist, it will be generated
func (cfg *ServerConfig) LoadConfig() (err error) {
	if cfg.CaCert == nil && cfg.CaCertFile != "" {
		cfg.CaCert, err = certs.LoadX509CertFromPEM(cfg.CaCertFile)
	}
	if err != nil {
		return err
	}
	if cfg.ServerCert == nil && cfg.ServerCertFile != "" {
		cfg.ServerCert, err = certs.LoadTLSCertFromPEM(cfg.ServerCertFile, cfg.ServerKeyFile)
	}
	if err != nil {
		return err
	}
	// load a key if a keyfile name was given
	if cfg.AppAccountKey == nil {
		if cfg.AppAccountKeyFile != "" {
			data, err := os.ReadFile(cfg.AppAccountKeyFile)
			if err != nil {
				return err
			}
			cfg.AppAccountKey, err = nkeys.ParseDecoratedNKey(data)
			if err != nil {
				return err
			}
		} else {
			cfg.AppAccountKey, err = nkeys.CreateAccount()
		}
	}
	return err
}

// NewServerConfig creates a new server configuration with default values
//
//	accountName is the application account name to use or "" for default
//	accountKey is the application account key or nil to get from config file or auto-generate
//	certsDir is the directory for loading certificate files, default is $HOME/certs
//	storesDir is the directory for storing hub data, default is $HOME/stores
//
// Default is $HOME/stores
func NewServerConfig(accountName string, accountKey nkeys.KeyPair, certsDir string, storesDir string) *ServerConfig {
	hostName, _ := os.Hostname()
	if accountName == "" {
		accountName = "hiveot-" + hostName
	}
	if accountKey == nil {
		accountKey, _ = nkeys.CreateAccount()
	}
	srvCfg := &ServerConfig{
		Host:           "127.0.0.1",
		Port:           4222,
		WSPort:         4223,
		StoresDir:      path.Join(storesDir, "server"),
		ServerKeyFile:  path.Join(certsDir, "serverKey.pem"),
		ServerCertFile: path.Join(certsDir, "serverCert.pem"),
		CaCertFile:     path.Join(certsDir, "caCert.pem"),
		AppAccountName: accountName,
		AppAccountKey:  accountKey,
	}
	return srvCfg
}
