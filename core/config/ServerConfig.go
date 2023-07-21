package config

import (
	"github.com/hiveot/hub/lib/certs"
	"os"
	"path"
)

// ServerConfig configuration of the messaging server
type ServerConfig struct {
	// Host is the server Hostname must match the name in the server certificate
	Host string `yaml:"host"`
	// Port, default is 4222
	Port   int `yaml:"port"`
	WSPort int `yaml:"WSPort"`
	//
	LogLevel string `yaml:"logLevel"`
	LogFile  string `yaml:"logFile"`
	//
	ServerCertFile string `yaml:"serverCertFile"`
	ServerKeyFile  string `yaml:"serverKeyFile"`
	DataDir        string `yaml:"dataDir"`
	CaCertFile     string `yaml:"caCertFile"`
	CaKeyFile      string `yaml:"caKeyFile"`
	// application account name and key for issued tokens
	AppAccountName    string `yaml:"appAccountName"`
	AppAccountKeyFile string `yaml:"appAccountKeyFile"`

	NoAutoStart bool `yaml:"noAutoStart"`

	// Loaded by InitConfig
	//AppAccountKey nkeys.KeyPair `yaml:"-"`
	//CaCert        *x509.Certificate `yaml:"-"`
	//ServerCert *tls.Certificate `yaml:"-"`
}

// InitConfig ensures all fields are valid.
// Any values already set will be kept as-is.
//
//	certsDir path to the certificate directory with certs and keys
//	storesDir path to the storage root directory. Server will use 'server' subdir.
func (cfg *ServerConfig) InitConfig(certsDir string, storesDir string) (err error) {

	hostName, _ := os.Hostname()

	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 4222
	}
	if cfg.WSPort == 0 {
		cfg.WSPort = 8222
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "server"
	}
	if !path.IsAbs(cfg.DataDir) {
		cfg.DataDir = path.Join(storesDir, cfg.DataDir)
	}

	if cfg.CaCertFile == "" {
		cfg.CaCertFile = certs.DefaultCaCertFile
	}
	if !path.IsAbs(cfg.ServerCertFile) {
		cfg.CaCertFile = path.Join(certsDir, cfg.CaCertFile)
	}

	if cfg.CaKeyFile == "" {
		cfg.CaKeyFile = certs.DefaultCaKeyFile
	}
	if !path.IsAbs(cfg.CaKeyFile) {
		cfg.CaKeyFile = path.Join(certsDir, cfg.CaKeyFile)
	}

	if cfg.ServerKeyFile == "" {
		cfg.ServerKeyFile = "serverKey.pem"
	}
	if !path.IsAbs(cfg.ServerKeyFile) {
		cfg.ServerKeyFile = path.Join(certsDir, cfg.ServerKeyFile)
	}

	if cfg.ServerCertFile == "" {
		cfg.ServerCertFile = "serverCert.pem"
	}
	if !path.IsAbs(cfg.ServerCertFile) {
		cfg.ServerCertFile = path.Join(certsDir, cfg.ServerCertFile)
	}

	if cfg.AppAccountName == "" {
		cfg.AppAccountName = "hiveot-" + hostName
	}
	if cfg.AppAccountKeyFile == "" {
		cfg.AppAccountKeyFile = cfg.AppAccountName + "Acct.nkey"
	}
	if !path.IsAbs(cfg.AppAccountKeyFile) {
		cfg.AppAccountKeyFile = path.Join(certsDir, cfg.AppAccountKeyFile)
	}

	return err
}
