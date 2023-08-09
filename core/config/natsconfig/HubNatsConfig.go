package natsconfig

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubNatsConfig with Hub core configuration
// Use NewHubNatsConfig to create a default config
type HubNatsConfig struct {
	// The home directory used in init and setup
	HomeDir string                 `json:"homeDir,omitempty"`
	Server  msgserver.ServerConfig `yaml:"server"`
	Authn   authn.AuthnConfig      `yaml:"authn"`
	Authz   authz.AuthzConfig      `yaml:"authz"`
}

// SetupCerts creates and loads CA and server certificates
func (cfg *HubNatsConfig) SetupCerts(new bool) (
	serverTLS *tls.Certificate,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
) {
	var alwaysNewServerCert = false
	var err error

	// 1: Load or create the CA certificate
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
		slog.Info("creating new CA certificate and key")
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
	// 3: Create the Server private key
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
	// 4: Create the Server cert
	if _, err2 := os.Stat(cfg.Server.ServerCertFile); err2 != nil || new {
		slog.Info("Creating new server cert")
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
	// 5: Finally load the server TLS cert
	serverTLS, err = certs.LoadTLSCertFromPEM(cfg.Server.ServerCertFile, cfg.Server.ServerKeyFile)
	if err != nil {
		panic("Unable to load server TLS cert. Is it malformed?: " + err.Error())
	}
	return serverTLS, caCert, caKey
}

func (cfg *HubNatsConfig) SetupOperator() (opKP nkeys.KeyPair, opJWT string, err error) {

	// Load/Create operator key
	if _, err2 := os.Stat(cfg.Server.OperatorKeyFile); err2 != nil {
		slog.Warn("Creating operator key file: " + cfg.Server.OperatorKeyFile)
		opKP, _ = nkeys.CreateOperator()
		operatorSeed, _ := opKP.Seed()
		err = os.WriteFile(cfg.Server.OperatorKeyFile, operatorSeed, 0400)
	} else {
		operatorSeed, _ := os.ReadFile(cfg.Server.OperatorKeyFile)
		opKP, err = nkeys.FromSeed(operatorSeed)
	}
	if err != nil {
		err = fmt.Errorf("error creating appOperatorKey: %w", err)
		return
	}
	operatorPub, _ := opKP.PublicKey()
	operatorClaims := jwt.NewOperatorClaims(operatorPub)
	operatorClaims.Name = "hiveotop"
	operatorClaims.SigningKeys.Add(operatorPub)
	opJWT, _ = operatorClaims.Encode(opKP)

	return opKP, opJWT, err
}

func (cfg *HubNatsConfig) SetupAppAccount(opKP nkeys.KeyPair) (
	appKP nkeys.KeyPair, appJWT string, err error) {

	// App Account
	if _, err2 := os.Stat(cfg.Server.AccountKeyFile); err2 != nil {
		slog.Info("Creating server account key file: " + cfg.Server.AccountKeyFile)
		appKP, _ = nkeys.CreateAccount()
		accountSeed, _ := appKP.Seed()
		err = os.WriteFile(cfg.Server.AccountKeyFile, accountSeed, 0400)
	} else {
		accountSeed, _ := os.ReadFile(cfg.Server.AccountKeyFile)
		appKP, err = nkeys.FromSeed(accountSeed)
	}
	if err != nil {
		err = fmt.Errorf("error creating appAcctKey: %w", err)
		return
	}
	appAccountKeyPub, _ := appKP.PublicKey()
	appAccountClaims := jwt.NewAccountClaims(appAccountKeyPub)
	appAccountClaims.Name = "AppAccount"
	// Enabling JetStream requires setting storage limits
	appAccountClaims.Limits.JetStreamLimits.DiskStorage = 1024 * 1024 * 1024
	appAccountClaims.Limits.JetStreamLimits.MemoryStorage = 100 * 1024 * 1024
	appJWT, _ = appAccountClaims.Encode(opKP)
	return appKP, appJWT, err
}

// Setup creates and loads certificate and key files
// if new is false then re-use existing certificate and key files.
// if new is true then create a whole new empty environment in the home directory
func (cfg *HubNatsConfig) Setup(new bool) (
	serverTLS *tls.Certificate,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	operatorNKey nkeys.KeyPair,
	operatorJWT string,
	systemJWT string,
	appAccountNKey nkeys.KeyPair,
	appAccountJWT string,
	serviceNKey nkeys.KeyPair,
	serviceJWT string,
) {
	var err error

	slog.Info("running setup",
		slog.Bool("--new", new), slog.String("home", cfg.HomeDir))

	// 1: in a new environment, clear the home directory
	// This is very destructive!
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

	// 2: Create/Load the certificates
	serverTLS, caCert, caKey = cfg.SetupCerts(new)

	// 3: Make sure the server storage dir exists
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

	// 4: Setup operator, system and account key chain
	operatorNKey, operatorJWT, err = cfg.SetupOperator()
	if err != nil {
		panic(err)
	}

	// System account key and JWT claims
	systemAccountNKey, _ := nkeys.CreateAccount()
	systemAccountPub, _ := systemAccountNKey.PublicKey()
	systemAccountClaims := jwt.NewAccountClaims(systemAccountPub)
	systemAccountClaims.Name = "SYS"
	//systemAccountClaims.SigningKeys.Add(systemSigningPub)
	systemAccountJWT, err := systemAccountClaims.Encode(operatorNKey)
	if err != nil {
		panic("error creating systemAccountJWT: " + err.Error())
	}

	// App Account
	appAccountNKey, appAccountJWT, err = cfg.SetupAppAccount(operatorNKey)

	// 8: Load/Create key for core services
	if _, err2 := os.Stat(cfg.Server.ServiceKeyFile); err2 != nil {
		slog.Info("Creating core services auth key file: " + cfg.Server.ServiceKeyFile)
		serviceNKey, _ = nkeys.CreateUser()
		serviceSeed, _ := serviceNKey.Seed()
		err = os.WriteFile(cfg.Server.ServiceKeyFile, serviceSeed, 0400)
	} else {
		serviceSeed, _ := os.ReadFile(cfg.Server.ServiceKeyFile)
		serviceNKey, err = nkeys.FromSeed(serviceSeed)
	}
	if err != nil {
		panic("error creating appServiceKey: " + err.Error())
	}
	serviceNKeyPub, _ := serviceNKey.PublicKey()
	serviceClaims := jwt.NewUserClaims(serviceNKeyPub)
	serviceClaims.Name = "hiveot-core-service"
	serviceClaims.IssuerAccount, _ = appAccountNKey.PublicKey()
	serviceJWT, err = serviceClaims.Encode(appAccountNKey)
	if err != nil {
		panic(err)
	}
	// 9: authn directories
	if _, err2 := os.Stat(cfg.Authn.CertsDir); err2 != nil {
		err = os.MkdirAll(cfg.Authn.CertsDir, 0755)
	}
	if _, err2 := os.Stat(path.Base(cfg.Authn.PasswordFile)); err2 != nil && err == nil {
		err = os.MkdirAll(path.Base(cfg.Authn.PasswordFile), 0700)
	}
	if err != nil {
		panic("error creating authn directories: " + err.Error())
	}

	// 10: authz directories
	if _, err2 := os.Stat(cfg.Authz.GroupsDir); err2 != nil {
		err = os.MkdirAll(cfg.Authz.GroupsDir, 0700)
	}
	if err != nil {
		panic("error creating authz directory: " + err.Error())
	}
	slog.Info("setup completed successfully")
	return serverTLS, caCert, caKey, operatorNKey, operatorJWT, systemAccountJWT, appAccountNKey, appAccountJWT, serviceNKey, serviceJWT
}

// NewHubNatsConfig creates and initalizes a configuration for the hub server and core services.
// This ensures that all fields have valid default values.
//
//	home dir of the application home. Default is the parent of the application bin folder
//	configfile with the name of the config file or "" to not load a config
func NewHubNatsConfig(homeDir string, configFile string) (*HubNatsConfig, error) {
	// default to the parent folder of the application binary
	if homeDir == "" {
		homeDir = path.Base(path.Base(os.Args[0]))
	}
	f := svcconfig.GetFolders(homeDir, false)
	hubCfg := &HubNatsConfig{HomeDir: f.Home}
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
