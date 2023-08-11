package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
)

// NatsJWTConfig holds the server configuration for use with JWT
type NatsJWTConfig struct {
	Host            string            // default localhost
	Port            int               // default 4222
	MaxDataMemoryMB int               // default 1GB
	DataDir         string            // default is server default
	ServerCert      *tls.Certificate  // default none
	CaCert          *x509.Certificate // default none
	SystemAccountKP nkeys.KeyPair     // default generate a new nkey
	AppAccountName  string            // default is hiveot
	AppAccountKP    nkeys.KeyPair     // default generate a new nkey
	AppServiceKP    nkeys.KeyPair     // default generate a new nkey
	// jwt specific
	OperatorKP       nkeys.KeyPair // default generate a new nkey
	OperatorJWT      string        // default generate a new JWT using key
	SystemAccountJWT string        // default generate a new JWT using key
	AppAccountJWT    string        // default generate a new JWT using key
	AppServiceJWT    string        // default generate a new JWT using key
}

// CreateNatsJWTOptions create a Nats options struct for use with JWT.
// This applies sensible defaults to cfg and returns the derived Nats and updated app cfg.
//
//	appCfg contains the application configurable parameters
func CreateNatsJWTOptions(appCfg NatsJWTConfig) (server.Options, NatsJWTConfig) {
	var err error

	// apply defaults as needed
	if appCfg.Host == "" {
		appCfg.Host = "localhost"
	}
	if appCfg.Port == 0 {
		appCfg.Port = 4222
	}
	if appCfg.MaxDataMemoryMB == 0 {
		appCfg.MaxDataMemoryMB = 1024
	}
	// ensure keys exist
	if appCfg.OperatorKP == nil {
		appCfg.OperatorKP, _ = nkeys.CreateOperator()
	}
	if appCfg.SystemAccountKP == nil {
		appCfg.SystemAccountKP, _ = nkeys.CreateAccount()
	}
	if appCfg.AppAccountName == "" {
		appCfg.AppAccountName = "hiveot"
	}
	if appCfg.AppAccountKP == nil {
		appCfg.AppAccountKP, _ = nkeys.CreateAccount()
	}
	if appCfg.AppServiceKP == nil {
		appCfg.AppServiceKP, _ = nkeys.CreateUser()
	}
	// create JWT tokens for each use
	if appCfg.OperatorJWT == "" {
		operatorPub, _ := appCfg.OperatorKP.PublicKey()
		operatorClaims := jwt.NewOperatorClaims(operatorPub)
		operatorClaims.Name = "hiveotop"
		// operator is self signed
		appCfg.OperatorJWT, _ = operatorClaims.Encode(appCfg.OperatorKP)
	}
	if appCfg.SystemAccountJWT == "" {
		systemAccountPub, _ := appCfg.SystemAccountKP.PublicKey()
		claims := jwt.NewAccountClaims(systemAccountPub)
		claims.Name = "SYS"
		appCfg.SystemAccountJWT, _ = claims.Encode(appCfg.OperatorKP)
	}
	if appCfg.AppAccountJWT == "" {
		appAccountPub, _ := appCfg.AppAccountKP.PublicKey()
		claims := jwt.NewAccountClaims(appAccountPub)
		claims.Name = appCfg.AppAccountName
		appCfg.AppAccountJWT, _ = claims.Encode(appCfg.OperatorKP)
	}
	if appCfg.AppServiceJWT == "" {
		appServicePub, _ := appCfg.AppServiceKP.PublicKey()
		claims := jwt.NewUserClaims(appServicePub)
		claims.Name = "HiveOTCoreService"
		claims.Tags.Add("clientType", authn.ClientTypeUser)
		appCfg.AppServiceJWT, err = claims.Encode(appCfg.AppAccountKP)
		_ = err
	}

	// Use of JWT requires an account resolver.
	// Use the simple in-memory resolver as there is only one account.
	memoryResolver := &server.MemAccResolver{}
	operatorClaims, err := jwt.DecodeOperatorClaims(appCfg.OperatorJWT)
	if err != nil {
		panic(err)
	}

	//systemClaims, _ := jwt.DecodeAccountClaims(appCfg.SystemAccountJWT)
	//systemAccountPub := systemClaims.Subject
	systemAcctPub, _ := appCfg.SystemAccountKP.PublicKey()
	err = memoryResolver.Store(systemAcctPub, appCfg.SystemAccountJWT)
	if err != nil {
		panic(err)
	}
	appAccountPub, _ := appCfg.AppAccountKP.PublicKey()
	_ = memoryResolver.Store(appAccountPub, appCfg.AppAccountJWT)
	if err != nil {
		panic(err)
	}
	appServiceKeyPub, _ := appCfg.AppServiceKP.PublicKey()
	_ = appServiceKeyPub
	//noAuthKey, _ := nkeys.FromSeed([]byte(natshubclient.PublicUnauthenticatedNKey))

	//noAuthKeyPub, _ := noAuthKey.PublicKey()
	natsOpts := server.Options{
		Host:             appCfg.Host,
		Port:             appCfg.Port,
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
		JetStreamMaxMemory: int64(appCfg.MaxDataMemoryMB) * 1024 * 1024,
		StoreDir:           appCfg.DataDir,

		// logging
		Debug:   true,
		Logtime: true,
	}

	if appCfg.CaCert != nil && appCfg.ServerCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(appCfg.CaCert)
		clientCertList := []tls.Certificate{*appCfg.ServerCert}
		tlsConfig := &tls.Config{
			ServerName:   "HiveOT Hub",
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}
		natsOpts.TLSTimeout = 1000 // for debugging auth
		natsOpts.TLSConfig = tlsConfig
	}
	return natsOpts, appCfg
}
