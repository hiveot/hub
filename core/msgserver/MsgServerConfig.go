package msgserver

import (
	"github.com/hiveot/hub/lib/certs"
	"path"
)

// MsgServerConfig configuration of the messaging server
type MsgServerConfig struct {
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
	//AppAccountName    string `yaml:"appAccountName"`
	//AppAccountKeyFile string `yaml:"appAccountKeyFile"`

	// The operator key signs the account key
	OperatorKeyFile string `yaml:"operatorKeyFile,omitempty"`
	// The account key signs the JWT token
	AccountKeyFile string `yaml:"accountKeyFile,omitempty"`
	// The service key is intended for use by services to connect
	ServiceKeyFile string `yaml:"serviceKeyFile,omitempty"`

	NoAutoStart bool `yaml:"noAutoStart"`

	// Maximum data memory RAM usage. 0 for default 100MB
	MaxDataMemoryMB int `json:"maxDataMemoryMB"`
}

// InitConfig ensures all fields are valid.
// Any values already set will be kept as-is.
//
//	certsDir path to the certificate directory with certs and keys
//	storesDir path to the storage root directory. Server will use 'server' subdir.
func (cfg *MsgServerConfig) InitConfig(certsDir string, storesDir string) (err error) {

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

	if cfg.MaxDataMemoryMB == 0 {
		cfg.MaxDataMemoryMB = 100
	}

	// operator sign jwt
	if cfg.OperatorKeyFile == "" {
		cfg.OperatorKeyFile = "operator.nkey"
	}
	if !path.IsAbs(cfg.OperatorKeyFile) {
		cfg.OperatorKeyFile = path.Join(certsDir, cfg.OperatorKeyFile)
	}

	// account signing key
	if cfg.AccountKeyFile == "" {
		cfg.AccountKeyFile = "appaccount.nkey"
	}
	if !path.IsAbs(cfg.AccountKeyFile) {
		cfg.AccountKeyFile = path.Join(certsDir, cfg.AccountKeyFile)
	}
	// services login key
	if cfg.ServiceKeyFile == "" {
		cfg.ServiceKeyFile = "services.nkey"
	}
	if !path.IsAbs(cfg.ServiceKeyFile) {
		cfg.ServiceKeyFile = path.Join(certsDir, cfg.ServiceKeyFile)
	}
	return err
}
