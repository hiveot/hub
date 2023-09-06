package mqttmsgserver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/certs"
	"golang.org/x/exp/slog"
	"path"
)

// MqttServerConfig holds the mqtt broker configuration
type MqttServerConfig struct {
	// configurable settings
	Host   string `yaml:"host,omitempty"`   // default: localhost
	Port   int    `yaml:"port,omitempty"`   // default: 8441
	WSPort int    `yaml:"wsPort,omitempty"` // default: 0 (disabled)

	LogLevel string `yaml:"logLevel,omitempty"` // default: warn
	LogFile  string `yaml:"logFile,omitempty"`  // default: no logfile
	Debug    bool   `yaml:"debug,omitempty"`    // default: false

	AdminUserKeyFile  string `yaml:"adminUserKeyFile,omitempty"`  // default: admin.nkey
	SystemUserKeyFile string `yaml:"systemUserKeyFile,omitempty"` // default: systemUser.nkey

	// Disable running the embedded messaging server. Default False
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`

	// The certs and keys can be set directly or loaded from above files
	CaCert        *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey         *ecdsa.PrivateKey `yaml:"-"` // preset, load, or error
	ServerTLS     *tls.Certificate  `yaml:"-"` // preset, load, or generate
	AdminUserKP   *ecdsa.PrivateKey `yaml:"-"` // generated
	CoreServiceKP *ecdsa.PrivateKey `yaml:"-"` // generated

	// The following options are JWT specific
	SystemAccountJWT string `yaml:"-"` // generated
	CoreServiceJWT   string `yaml:"-"` // generated
}

// Setup the nats server config.
// This applies sensible defaults to Config.
//
// Any existing values that are previously set remain unchanged.
// Missing values are created.
// Certs and keys are loaded as per configuration.
//
// Set 'writeChanges' to persist generated server cert, operator and account keys
//
//	certsDir is the default certificate and key location
//	storesDir is the data storage root (default $HOME/stores)
func (cfg *MqttServerConfig) Setup(certsDir, storesDir string, writeChanges bool) (err error) {

	// Step 1: Apply defaults parameters
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 4222
	}
	if cfg.WSPort == 0 {
		//appCfg.WSPort = 8222
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}
	if cfg.AdminUserKeyFile == "" {
		cfg.AdminUserKeyFile = path.Join(certsDir, "admin.pem")
	}
	if cfg.SystemUserKeyFile == "" {
		cfg.SystemUserKeyFile = path.Join(certsDir, "systemUser.pem")
	}

	// Step 2: generate missing certificates
	// These are typically set directly before running setup so this is intended
	// for testing.
	if cfg.CaCert == nil || cfg.CaKey == nil {
		cfg.CaCert, cfg.CaKey, err = certs.CreateCA("hiveot", 365)
	}
	if cfg.ServerTLS == nil && cfg.CaKey != nil {
		serverKeys := certs.CreateECDSAKeys()
		names := []string{cfg.Host}
		serverX509, err := certs.CreateServerCert(
			"hiveot", "server",
			365, // validity matches the CA
			&serverKeys.PublicKey,
			names, cfg.CaCert, cfg.CaKey)
		if err != nil {
			slog.Error("unable to generate server cert. Not using TLS.", "err", err)
		} else {
			cfg.ServerTLS = certs.X509CertToTLS(serverX509, serverKeys)
		}
	}

	// Step 4: generate derived keys
	if cfg.AdminUserKP == nil {
		cfg.AdminUserKP, _ = cfg.LoadCreateUserKP(cfg.AdminUserKeyFile, writeChanges)
	}
	if cfg.CoreServiceKP == nil {
		cfg.CoreServiceKP, _ = cfg.LoadCreateUserKP(cfg.CoreServiceKeyFile, writeChanges)
	}
	if cfg.AdminUserKeyFile != "" && writeChanges {
		cfg.AdminUserKP, _ = cfg.LoadCreateUserKP(cfg.AdminUserKeyFile, writeChanges)
	}
	// Step 5: generate the JWT tokens -
	// disables as callouts are stable
	//if cfg.OperatorJWT == "" {
	//	operatorPub, _ := cfg.OperatorKP.PublicKey()
	//	operatorClaims := jwt.NewOperatorClaims(operatorPub)
	//	operatorClaims.Name = "hiveotop"
	//	// operator is self signed
	//	cfg.OperatorJWT, err = operatorClaims.Encode(cfg.OperatorKP)
	//	if err != nil {
	//		return fmt.Errorf("OperatorJWT error: %w", err)
	//	}
	//}
	//if cfg.SystemAccountJWT == "" {
	//	systemAccountPub, _ := cfg.SystemAccountKP.PublicKey()
	//	claims := jwt.NewAccountClaims(systemAccountPub)
	//	claims.Name = "$SYS"
	//	cfg.SystemAccountJWT, err = claims.Encode(cfg.OperatorKP)
	//	if err != nil {
	//		return fmt.Errorf("SystemAccountJWT error: %w", err)
	//	}
	//}
	//if cfg.AppAccountJWT == "" {
	//	appAccountPub, _ := cfg.AppAccountKP.PublicKey()
	//	claims := jwt.NewAccountClaims(appAccountPub)
	//	claims.Name = cfg.AppAccountName
	//	claims.Limits.JetStreamLimits.DiskStorage = -1
	//	claims.Limits.JetStreamLimits.MemoryStorage = int64(cfg.MaxDataMemoryMB) * 1024 * 1024
	//	cfg.AppAccountJWT, err = claims.Encode(cfg.OperatorKP)
	//	if err != nil {
	//		return fmt.Errorf("AppAccountJWT error: %w", err)
	//	}
	//}
	//if cfg.CoreServiceJWT == "" {
	//	coreServicePub, _ := cfg.CoreServiceKP.PublicKey()
	//	claims := jwt.NewUserClaims(coreServicePub)
	//	claims.Name = "HiveOTCoreService"
	//	claims.Tags.Add("clientType", auth.ClientTypeUser)
	//	cfg.CoreServiceJWT, err = claims.Encode(cfg.AppAccountKP)
	//	if err != nil {
	//		return fmt.Errorf("CoreServiceJWT error: %w", err)
	//	}
	//}
	return nil
}
