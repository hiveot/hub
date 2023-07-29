package hubnats

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/core/config"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>", "_INBOX.>"}},
}

// Authn service permissions
var authnPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"things.authn.*.action.>"}},
}

// Authz service permissions
var authzPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"things.authz.*.action.>"}},
}

//var adminPermissions = &server.Permissions{
//	Publish:   &server.SubjectPermission{Allow: []string{">"}},
//	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
//}

// NatsConfigurator manages the nats server configuration
// This implements INatsConfigurator
type NatsConfigurator struct {
	// hub server configuration
	hubcfg *config.ServerConfig
	// the nats server static configuration
	natsOpts *server.Options
	// nats server to whose configuration to manage
	ns *server.Server
}

// AddServiceKey adds a user nkey to the server config on the application account,
// and exclude it from the callout account.
// Intended to work around having implement nkey auth ourselves.
// This takes effect immediately.
func (svc *NatsConfigurator) AddServiceKey(nkeyPub string) error {
	//slog.Info("Adding service key for account")
	//appAcct, err := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	//var err error
	//var appAcct *server.Account // not used
	//if err != nil {
	//	return fmt.Errorf("app account not set")
	//}
	//if srv.chook == nil {
	svc.natsOpts.Nkeys = append(svc.natsOpts.Nkeys, &server.NkeyUser{
		Nkey: nkeyPub,
		//Account: appAcct,
		//Permissions: p,
	})
	//} else {
	//	err = srv.chook.AddServiceKey(nkeyPub, appAcct)
	//}
	//if err == nil {
	err := svc.ns.ReloadOptions(svc.natsOpts)
	//}
	return err
}

// GetServerOpts returns the current server configuration
func (svc *NatsConfigurator) GetServerOpts() *server.Options {
	return svc.natsOpts
}

// InitCalloutHook initializes use of nats callout using the given verification handler
// This must be invoked after starting the server as it needs the application account.
//func (srv *HubNatsServer) InitCalloutHook(
//	authnVerifier func(request *jwt.AuthorizationRequestClaims) error) error {
//
//	slog.Info("InitCalloutHook")
//	// create a server connection for use by the callout handler. Use a temporary key.
//	serviceKey, _ := nkeys.CreateUser()
//	serviceKeyPub, _ := serviceKey.PublicKey()
//	err := srv.AddServiceKey(serviceKeyPub)
//	if err != nil {
//		return err
//	}
//	err = srv.ns.ReloadOptions(srv.serverOpts)
//	if err != nil {
//		return err
//	}
//	serviceConn, err := srv.ConnectInProc("CalloutHook", serviceKey)
//	if err != nil {
//		return err
//	}
//
//	// install the callout handler
//	srv.chook, err = ConnectNatsCalloutHook(
//		srv.serverOpts,
//		srv.cfg.AppAccountName,
//		srv.appAcctKey,
//		serviceConn,
//		authnVerifier,
//	)
//
//	return err
//}

// Start applies the configuration to the server
// This allows the reload of options after changes, as needed
func (svc *NatsConfigurator) Start(ns *server.Server) {
	svc.ns = ns
	//svc.ns.ConfigureLogger()
}

// NewNatsConfigurator create the instance for managing the NATS server configuration
// This creates a configuration based on the parameters. Use 'Start' to set the nats server.
func NewNatsConfigurator(
	hubcfg *config.ServerConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	operatorJWT string,
	systemAccountJWT string,
	appAccountJWT string,
	serviceKey nkeys.KeyPair,
) *NatsConfigurator {

	// JWT needs an account resolver.
	// Use the simple in-memory resolver as there is only one account.
	memoryResolver := &server.MemAccResolver{}
	operatorClaim, err := jwt.DecodeOperatorClaims(operatorJWT)
	if err != nil {
		panic(err)
	}

	systemClaims, _ := jwt.DecodeAccountClaims(systemAccountJWT)
	systemAccountPub := systemClaims.Subject
	err = memoryResolver.Store(systemClaims.Subject, systemAccountJWT)
	if err != nil {
		panic(err)
	}
	appAccountClaims, _ := jwt.DecodeAccountClaims(appAccountJWT)
	_ = memoryResolver.Store(appAccountClaims.Subject, appAccountJWT)
	if err != nil {
		panic(err)
	}
	serviceKeyPub, _ := serviceKey.PublicKey()
	_ = serviceKeyPub
	//systemAccountPub, err := systemAccountNKey.PublicKey()
	natsOpts := &server.Options{
		Host:             hubcfg.Host,
		Port:             hubcfg.Port,
		AccountResolver:  memoryResolver,
		SystemAccount:    systemAccountPub, //"SYS",
		TrustedOperators: []*jwt.OperatorClaims{operatorClaim},

		JetStream:          true,
		JetStreamMaxMemory: int64(hubcfg.MaxDataMemoryMB) * 1024 * 1024,

		// logging
		Debug:   true,
		Logtime: true,
	}

	if caCert != nil && serverCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(caCert)
		clientCertList := []tls.Certificate{*serverCert}
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
	svc := &NatsConfigurator{
		hubcfg:   hubcfg,
		natsOpts: natsOpts,
	}
	return svc
}
