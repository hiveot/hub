package natsserver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"os"
	"path"
)

const NoAuthUserID = "unauthenticated"

// NatsServerConfig holds the configuration for nkeys and jwt based servers
type NatsServerConfig struct {
	// configurable settings
	Host            string `yaml:"host"`            // default: localhost
	Port            int    `yaml:"port"`            // default: 4222
	WSPort          int    `yaml:"wsPort"`          // default: 0 (disabled)
	MaxDataMemoryMB int    `yaml:"maxDataMemoryMB"` // default: 1024
	DataDir         string `yaml:"dataDir"`         // default is server default
	AppAccountName  string `yaml:"appAccountName"`  // default: hiveot
	Debug           bool   `yaml:"debug"`           // default: false
	LogLevel        string `yaml:"logLevel"`        // default: warn
	LogFile         string `yaml:"logFile"`         // default: no logfile

	// optional files that persist cert and keys
	CaCertFile     string `yaml:"caCertFile"`     // default: caCert.pem
	CaKeyFile      string `yaml:"caKeyFile"`      // default: caKey.pem
	ServerCertFile string `yaml:"serverCertFile"` // default: hubCert.pem
	ServerKeyFile  string `yaml:"serverKeyFile"`  // default: kubKey.pem
	//
	OperatorKeyFile   string `yaml:"operatorKeyFile"`   // default: operator.nkey (jwt only)
	AppAccountKeyFile string `yaml:"appAccountKeyFile"` // default: appAcct.nkey

	// The certs and keys can be set directly or loaded from above files
	CaCert          *x509.Certificate `yaml:"-"` // preset, load, or error
	CaKey           *ecdsa.PrivateKey `yaml:"-"` // preset, load, or error
	ServerCert      *tls.Certificate  `yaml:"-"` // preset, load, or generate
	AppAccountKP    nkeys.KeyPair     `yaml:"-"` // preset, load, or generate
	SystemAccountKP nkeys.KeyPair     `yaml:"-"` // generated
	CoreServiceKP   nkeys.KeyPair     `yaml:"-"` // generated

	// The following options are JWT specific
	OperatorKP       nkeys.KeyPair `yaml:"-"` // loaded or generated
	OperatorJWT      string        `yaml:"-"` // generated from OperatorKP
	SystemAccountJWT string        `yaml:"-"` // generated
	AppAccountJWT    string        `yaml:"-"` // generated from AppAccountKP
	CoreServiceJWT   string        `yaml:"-"` // generated
}

// Setup the nats server config.
// This applies sensible defaults to cfg.
//
// Any existing values that are previously set remain unchanged.
// Missing values are created.
// Certs and keys are loaded as per configuration.
//
// Set 'writeChanges' to persist generated server cert, operator and account keys
//
//	certsDir is the default certificate location
//	storesDir is the data storage root (default $HOME/stores)
func (cfg *NatsServerConfig) Setup(certsDir, storesDir string, writeChanges bool) error {
	var err error

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
	if cfg.DataDir == "" {
		cfg.DataDir = path.Join(storesDir, "natsserver")
	}
	if cfg.MaxDataMemoryMB == 0 {
		cfg.MaxDataMemoryMB = 1024
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}
	// file
	if cfg.CaCertFile == "" {
		cfg.CaCertFile = certs.DefaultCaCertFile
	}
	if cfg.CaKeyFile == "" {
		cfg.CaKeyFile = certs.DefaultCaKeyFile
	}
	if cfg.ServerCertFile == "" {
		cfg.ServerCertFile = "hubCert.pem"
	}
	if cfg.ServerKeyFile == "" {
		cfg.ServerKeyFile = "hubKey.pem"
	}
	if cfg.AppAccountName == "" {
		cfg.AppAccountName = "hiveot"
	}
	if cfg.AppAccountKeyFile == "" {
		cfg.AppAccountKeyFile = cfg.AppAccountName + ".nkey"
	}
	if cfg.OperatorKeyFile == "" {
		cfg.OperatorKeyFile = "operator.nkey"
	}

	// Step 2: load or generate missing certificates
	if cfg.CaCert == nil {
		caCertPath := cfg.CaCertFile
		if !path.IsAbs(caCertPath) {
			caCertPath = path.Join(certsDir, caCertPath)
		}
		cfg.CaCert, err = certs.LoadX509CertFromPEM(caCertPath)
		if err != nil {
			slog.Warn("missing CA cert. Continue without TLS.", "err", err)
		}
	}
	// only load the ca key if the cert was loaded
	if cfg.CaCert != nil && cfg.CaKey == nil {
		caKeyPath := cfg.CaKeyFile
		if !path.IsAbs(caKeyPath) {
			caKeyPath = path.Join(certsDir, caKeyPath)
		}
		cfg.CaKey, err = certs.LoadKeysFromPEM(caKeyPath)
		if err != nil {
			return fmt.Errorf("missing CA key: %w", err)
		}
	}
	// without a server cert TLS is not used
	if cfg.ServerCert == nil && cfg.CaKey != nil {
		serverCertPath := cfg.ServerCertFile
		serverKeyPath := cfg.ServerKeyFile
		if !path.IsAbs(serverCertPath) {
			serverCertPath = path.Join(certsDir, serverCertPath)
		}
		if !path.IsAbs(serverKeyPath) {
			serverKeyPath = path.Join(certsDir, serverKeyPath)
		}
		cfg.ServerCert, err = certs.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)
		// if file can't be loaded then generate it on the fly
		if err != nil {
			serverKeys := certs.CreateECDSAKeys()
			names := []string{cfg.Host}
			serverX509, err := certs.CreateServerCert(
				cfg.AppAccountName, "server", certs.DefaultServerCertValidityDays,
				&serverKeys.PublicKey,
				names, cfg.CaCert, cfg.CaKey)
			if err != nil {
				slog.Error("unable to generate server cert. Not using TLS.", "err", err)
			} else {
				cfg.ServerCert = certs.X509CertToTLS(serverX509, serverKeys)
				if writeChanges {
					err = certs.SaveTLSCertToPEM(cfg.ServerCert, serverCertPath, serverKeyPath)
					if err != nil {
						slog.Error("failed to persist server cert", "err", err)
					}
				}
			}
		}
	}

	// Step 3: Load or generate Account key

	//if appCfg.SystemAccountKP == nil {
	//	appCfg.SystemAccountKP, _ = nkeys.CreateAccount()
	//}
	// if not preset,load or generate the operator key
	if cfg.OperatorKP == nil {
		kpPath := cfg.OperatorKeyFile
		if !path.IsAbs(kpPath) {
			kpPath = path.Join(certsDir, kpPath)
		}
		kpSeed, err := os.ReadFile(kpPath)
		if err == nil {
			cfg.OperatorKP, err = nkeys.ParseDecoratedNKey(kpSeed)
		}
		if err != nil {
			slog.Warn("Generating operator key.")
			cfg.OperatorKP, _ = nkeys.CreateOperator()
			if writeChanges {
				kpSeed, _ := cfg.OperatorKP.Seed()
				err = os.WriteFile(kpPath, kpSeed, 0400)
				if err != nil {
					return fmt.Errorf("failed to persist operator key: %w", err)
				}
			}
		}
	}
	// if not preset,load or generate the operator key
	if cfg.AppAccountKP == nil {
		kpPath := cfg.AppAccountKeyFile
		if !path.IsAbs(kpPath) {
			kpPath = path.Join(certsDir, kpPath)
		}
		kpSeed, err := os.ReadFile(kpPath)
		if err == nil {
			cfg.AppAccountKP, err = nkeys.ParseDecoratedNKey(kpSeed)
		}
		if err != nil {
			slog.Warn("Generating app account key.")
			cfg.AppAccountKP, _ = nkeys.CreateAccount()
			if writeChanges {
				kpSeed, _ := cfg.AppAccountKP.Seed()
				err = os.WriteFile(kpPath, kpSeed, 0400)
				if err != nil {
					return fmt.Errorf("failed to persist app account key: %w", err)
				}
			}
		}
	}

	// Step 4: generate derived keys
	if cfg.SystemAccountKP == nil {
		cfg.SystemAccountKP, _ = nkeys.CreateAccount()
	}
	if cfg.CoreServiceKP == nil {
		cfg.CoreServiceKP, _ = nkeys.CreateUser()
	}

	// Step 5: generate the JWT tokens
	if cfg.OperatorJWT == "" {
		operatorPub, _ := cfg.OperatorKP.PublicKey()
		operatorClaims := jwt.NewOperatorClaims(operatorPub)
		operatorClaims.Name = "hiveotop"
		// operator is self signed
		cfg.OperatorJWT, err = operatorClaims.Encode(cfg.OperatorKP)
		if err != nil {
			return fmt.Errorf("OperatorJWT error: %w", err)
		}
	}
	if cfg.SystemAccountJWT == "" {
		systemAccountPub, _ := cfg.SystemAccountKP.PublicKey()
		claims := jwt.NewAccountClaims(systemAccountPub)
		claims.Name = "$SYS"
		cfg.SystemAccountJWT, err = claims.Encode(cfg.OperatorKP)
		if err != nil {
			return fmt.Errorf("SystemAccountJWT error: %w", err)
		}
	}
	if cfg.AppAccountJWT == "" {
		appAccountPub, _ := cfg.AppAccountKP.PublicKey()
		claims := jwt.NewAccountClaims(appAccountPub)
		claims.Name = cfg.AppAccountName
		claims.Limits.JetStreamLimits.DiskStorage = -1
		claims.Limits.JetStreamLimits.MemoryStorage = int64(cfg.MaxDataMemoryMB) * 1024 * 1024
		cfg.AppAccountJWT, err = claims.Encode(cfg.OperatorKP)
		if err != nil {
			return fmt.Errorf("AppAccountJWT error: %w", err)
		}
	}
	if cfg.CoreServiceJWT == "" {
		coreServicePub, _ := cfg.CoreServiceKP.PublicKey()
		claims := jwt.NewUserClaims(coreServicePub)
		claims.Name = "HiveOTCoreService"
		claims.Tags.Add("clientType", authn.ClientTypeUser)
		cfg.CoreServiceJWT, err = claims.Encode(cfg.AppAccountKP)
		if err != nil {
			return fmt.Errorf("CoreServiceJWT error: %w", err)
		}
	}
	return nil
}

// TODO use inbox prefix
// unauthenticated users are allowed to login and receive a token
var noAuthPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"things.authn.client.action.login.>"}, Deny: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.unauthenticated.>"}, Deny: []string{">"}},
}

//var adminPermissions = &server.Permissions{
//	Publish:   &server.SubjectPermission{Allow: []string{">"}},
//	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
//}
//
//var userPermissions = &server.Permissions{
//	Publish:   &server.SubjectPermission{Allow: []string{"things.>"}},
//	Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
//}

// CreateNatsNKeyOptions create a Nats options struct for use with NKey authentication.
// Note that Setup() must have been called first.
func (cfg *NatsServerConfig) CreateNatsNKeyOptions() server.Options {

	//systemAccountPub, _ := cfg.SystemAccountKP.PublicKey()
	appAccountPub, _ := cfg.AppAccountKP.PublicKey()
	//coreServiceKeyPub, _ := cfg.CoreServiceKP.PublicKey()

	//systemAcct := server.NewAccount("$SYS")
	//systemAcct.Nkey = systemAccountPub
	// NewAccount creates a limitless account. There is no way to set a limit though :/
	appAcct := server.NewAccount(cfg.AppAccountName)
	appAcct.Nkey = appAccountPub

	natsOpts := server.Options{
		Host: cfg.Host,
		Port: cfg.Port,

		//SystemAccount: "$SYS",
		Accounts: []*server.Account{appAcct},
		//Accounts:      []*server.Account{systemAcct, appAcct},
		NoAuthUser: NoAuthUserID,

		Nkeys: []*server.NkeyUser{},
		// login without password is needed to obtain a token using a password
		Users: []*server.User{
			{
				Username:    NoAuthUserID,
				Password:    "",
				Permissions: noAuthPermissions,
				Account:     appAcct,
				//InboxPrefix: "_INBOX." + NoAuthUserID,
			},
		},
		JetStream:          true,
		JetStreamMaxMemory: int64(cfg.MaxDataMemoryMB) * 1024 * 1024,
		StoreDir:           cfg.DataDir,

		// logging
		Debug:   cfg.Debug,
		Logtime: true,
	}

	if cfg.CaCert != nil && cfg.ServerCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cfg.CaCert)
		clientCertList := []tls.Certificate{*cfg.ServerCert}
		tlsConfig := &tls.Config{
			ServerName:   "HiveOT Hub",
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}
		natsOpts.AuthTimeout = 101 // for debugging auth
		natsOpts.TLSTimeout = 100  // for debugging auth
		natsOpts.TLSConfig = tlsConfig
	}
	return natsOpts
}

// CreateNatsJWTOptions create a Nats options struct for use with JWT.
// This applies sensible defaults to cfg and returns the derived Nats and updated app cfg.
// Note that Setup() must have been called first.
func (cfg *NatsServerConfig) CreateNatsJWTOptions() server.Options {
	var err error

	// Use of JWT requires an account resolver.
	// Use the simple in-memory resolver as there is only one account.
	memoryResolver := &server.MemAccResolver{}
	operatorClaims, err := jwt.DecodeOperatorClaims(cfg.OperatorJWT)
	if err != nil {
		panic(err)
	}

	//systemClaims, _ := jwt.DecodeAccountClaims(cfg.SystemAccountJWT)
	//systemAccountPub := systemClaims.Subject
	systemAcctPub, _ := cfg.SystemAccountKP.PublicKey()
	err = memoryResolver.Store(systemAcctPub, cfg.SystemAccountJWT)
	if err != nil {
		panic(err)
	}
	appAccountPub, _ := cfg.AppAccountKP.PublicKey()
	_ = memoryResolver.Store(appAccountPub, cfg.AppAccountJWT)
	if err != nil {
		panic(err)
	}
	appServiceKeyPub, _ := cfg.CoreServiceKP.PublicKey()
	_ = appServiceKeyPub
	//noAuthKey, _ := nkeys.FromSeed([]byte(natshubclient.PublicUnauthenticatedNKey))

	//noAuthKeyPub, _ := noAuthKey.PublicKey()
	natsOpts := server.Options{
		Host:             cfg.Host,
		Port:             cfg.Port,
		AccountResolver:  memoryResolver,
		SystemAccount:    systemAcctPub, //"SYS",
		TrustedOperators: []*jwt.OperatorClaims{operatorClaims},

		//TrustedKeys: []string{operatorClaim.Subject},
		//Nkeys: []*server.NkeyUser{
		//	{Nkey: serviceKeyPub},
		//	{Nkey: noAuthKeyPub}}, //Permissions:            noAuthPermissions,
		//Account:                appAccountClaims.Subject,
		//SigningKey:             "",
		//AllowedConnectionTypes: nil,

		JetStream:          true,
		JetStreamMaxMemory: int64(cfg.MaxDataMemoryMB) * 1024 * 1024,
		StoreDir:           cfg.DataDir,

		// logging
		Debug:   true,
		Logtime: true,
	}

	if cfg.CaCert != nil && cfg.ServerCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cfg.CaCert)
		clientCertList := []tls.Certificate{*cfg.ServerCert}
		tlsConfig := &tls.Config{
			ServerName:   "HiveOT Hub",
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}
		natsOpts.AuthTimeout = 101 // for debugging auth
		natsOpts.TLSTimeout = 100  // for debugging auth
		natsOpts.TLSConfig = tlsConfig
	}
	return natsOpts
}
