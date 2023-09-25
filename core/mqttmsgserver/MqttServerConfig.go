package mqttmsgserver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/mqttmsgserver/jwtauth"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"log/slog"
	"os"
	"path"
)

const DefaultAdminKeyFileName = "adminKey.pem"
const DefaultAdminTokenFileName = "adminToken.jwt"

// MqttServerConfig holds the mqtt broker configuration
type MqttServerConfig struct {
	// configurable settings
	Host   string `yaml:"host,omitempty"`   // default: localhost
	Port   int    `yaml:"port,omitempty"`   // 0 default: 8883
	WSPort int    `yaml:"wsPort,omitempty"` // 0 default: 8884

	LogLevel string `yaml:"logLevel,omitempty"` // default: warn
	LogFile  string `yaml:"logFile,omitempty"`  // default: no logfile
	Debug    bool   `yaml:"debug,omitempty"`    // default: false

	DataDir string `yaml:"dataDir,omitempty"` // default is server default

	AdminUserKeyFile   string `yaml:"adminUserKeyFile,omitempty"`   // default: adminKey.pem
	AdminUserTokenFile string `yaml:"adminUserTokenFile,omitempty"` // default: adminToken.jwt

	// Disable running the embedded messaging server. Default False
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`

	// the in-proc UDS name to use. Default is "@/MqttInMemUDSProd" (see MqttHubClient)
	InProcUDSName string `yaml:"inProcUDSName"`

	// The certs and keys can be set directly or loaded from above files
	CaCert    *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey     *ecdsa.PrivateKey `yaml:"-"` // preset, load, or error
	ServerKey *ecdsa.PrivateKey `yaml:"-"` // generated, loaded  (used as signing key)
	ServerTLS *tls.Certificate  `yaml:"-"` // generated

	AdminUserKP  *ecdsa.PrivateKey `yaml:"-"` // generated
	AdminUserPub string            `yaml:"-"` // generated

	CoreServiceKP  *ecdsa.PrivateKey `yaml:"-"` // generated
	CoreServicePub string            `yaml:"-"` // generated

	// The following options are JWT specific
}

// Setup the mqtt server config.
// This applies sensible defaults to Config.
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
		cfg.Host = "localhost"
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
	if cfg.AdminUserKeyFile == "" {
		cfg.AdminUserKeyFile = path.Join(keysDir, DefaultAdminKeyFileName)
	}
	if cfg.AdminUserTokenFile == "" {
		cfg.AdminUserTokenFile = path.Join(keysDir, DefaultAdminTokenFileName)
	}
	if cfg.InProcUDSName == "" {
		cfg.InProcUDSName = mqtthubclient.MqttInMemUDSProd
	}

	// Step 2: generate missing certificates
	// These are typically set directly before running setup so this is intended
	// for testing.
	if cfg.CaCert == nil || cfg.CaKey == nil {
		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365)
	}
	if cfg.ServerKey == nil {
		cfg.ServerKey, _ = certs.CreateECDSAKeys()
	}
	if cfg.ServerTLS == nil && cfg.CaKey != nil {
		names := []string{cfg.Host}
		serverX509, err := certs.CreateServerCert(
			"hiveot", "server",
			365, // validity matches the CA
			&cfg.ServerKey.PublicKey,
			names, cfg.CaCert, cfg.CaKey)
		if err != nil {
			slog.Error("unable to generate server cert. Not using TLS.", "err", err)
		} else {
			cfg.ServerTLS = certs.X509CertToTLS(serverX509, cfg.ServerKey)
		}
	}

	// Step 4: generate admin keys and token
	// core service keys are always regenerated and not saved
	if cfg.CoreServiceKP == nil {
		cfg.CoreServiceKP, cfg.CoreServicePub = certs.CreateECDSAKeys()
	}
	// admin user might need the key for hubcli
	if cfg.AdminUserKP == nil {
		cfg.AdminUserKP, cfg.AdminUserPub, err = jwtauth.LoadCreateUserKP(cfg.AdminUserKeyFile, writeChanges)
		if err != nil {
			slog.Error(err.Error())
		}
	}

	// make sure the admin auth token exists
	if _, err = os.Stat(cfg.AdminUserTokenFile); err != nil {
		adminToken, _ := jwtauth.CreateToken(msgserver.ClientAuthInfo{
			ClientID:   "admin",
			ClientType: auth.ClientTypeUser,
			PubKey:     cfg.AdminUserPub,
			Role:       auth.ClientRoleAdmin}, cfg.ServerKey)
		_ = os.MkdirAll(path.Dir(cfg.AdminUserTokenFile), 0700)
		err = os.WriteFile(cfg.AdminUserTokenFile, []byte(adminToken), 0400)
		if err != nil {
			slog.Error(err.Error())
		}

	}
	return nil
}
