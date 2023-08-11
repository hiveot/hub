package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
)

const NoAuthUserID = "unauthenticated"

// NatsNKeysConfig holds the server configuration for use with NKeys based authentication
type NatsNKeysConfig struct {
	Host            string            // default localhost
	Port            int               // default 4222
	MaxDataMemoryMB int               // default 1GB
	DataDir         string            // default is server default
	ServerCert      *tls.Certificate  // default none
	CaCert          *x509.Certificate // default none
	SystemAccountKP nkeys.KeyPair     // default generate a new nkey
	AppAccountName  string            // default is hiveot
	AppAccountKP    nkeys.KeyPair     // default generate a new nkey
	CoreServiceKP   nkeys.KeyPair     // default generate a new nkey
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
// This applies sensible defaults to cfg and returns the derived Nats and app cfg.
//
//	appCfg contains the application configurable parameters
func CreateNatsNKeyOptions(appCfg NatsNKeysConfig) (server.Options, NatsNKeysConfig) {

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
	//if appCfg.SystemAccountKP == nil {
	//	appCfg.SystemAccountKP, _ = nkeys.CreateAccount()
	//}
	if appCfg.AppAccountName == "" {
		appCfg.AppAccountName = "hiveot"
	}
	if appCfg.AppAccountKP == nil {
		appCfg.AppAccountKP, _ = nkeys.CreateAccount()
	}
	if appCfg.CoreServiceKP == nil {
		appCfg.CoreServiceKP, _ = nkeys.CreateUser()
	}

	//systemAccountPub, _ := appCfg.SystemAccountKP.PublicKey()
	appAccountPub, _ := appCfg.AppAccountKP.PublicKey()
	//coreServiceKeyPub, _ := appCfg.CoreServiceKP.PublicKey()

	//systemAcct := server.NewAccount("$SYS")
	//systemAcct.Nkey = systemAccountPub
	appAcct := server.NewAccount(appCfg.AppAccountName)
	appAcct.Nkey = appAccountPub

	natsOpts := server.Options{
		Host: appCfg.Host,
		Port: appCfg.Port,
		//SystemAccount: "$SYS",
		Accounts: []*server.Account{appAcct},
		//Accounts:      []*server.Account{systemAcct, appAcct},
		NoAuthUser: NoAuthUserID,

		Nkeys: []*server.NkeyUser{
			// pre-populate with the built-in core service key
			//{
			//	Nkey:    coreServiceKeyPub,
			//	Account: appAcct,
			//	// app services have full permissions
			//},
		},
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
