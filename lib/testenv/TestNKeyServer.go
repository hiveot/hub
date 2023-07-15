package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"time"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}, Deny: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>", "_INBOX.>"}, Deny: []string{">"}},
}

var adminPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
}

var userPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"things.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
}

// TestNKeyServer is an embedded NATS test messaging server using leaf node configuration
type TestNKeyServer struct {
	account    *server.Account
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	ns         *server.Server
	//
}

// Start the server, listening on 127.0.0.1
// setup accounts provided with the given bundle
func (svr *TestNKeyServer) Start(bundle TestAuthBundle) (clientURL string, err error) {
	operatorClaims, _ := jwt.DecodeOperatorClaims(bundle.OperatorJWT)
	_ = operatorClaims
	systemClaims, _ := jwt.DecodeAccountClaims(bundle.SystemAccountJWT)
	_ = systemClaims
	operatorNKeyPub, _ := bundle.OperatorNKey.PublicKey()
	_ = operatorNKeyPub
	systemAccountNKeyPub, _ := bundle.SystemAccountNKey.PublicKey()
	_ = systemAccountNKeyPub
	systemSigningNKeyPub, _ := bundle.SystemSigningNKey.PublicKey()
	_ = systemSigningNKeyPub
	appAccountNKeyPub, _ := bundle.AppAccountNKey.PublicKey()
	_ = appAccountNKeyPub
	appSigningNKeyPub, _ := bundle.AppSigningNKey.PublicKey()
	_ = appSigningNKeyPub
	serviceNKeyPub, _ := bundle.ServiceNKey.PublicKey()
	_ = serviceNKeyPub
	userNKeyPub, _ := bundle.UserNKey.PublicKey()
	_ = userNKeyPub

	//memoryResolver := &server.MemAccResolver{}
	//err = memoryResolver.Store(systemClaims.Subject, bundle.SystemAccountJWT)
	//if err != nil {
	//	panic(err)
	//}
	//appAccountClaims, _ := jwt.DecodeAccountClaims(bundle.AppAccountJWT)
	//_ = memoryResolver.Store(appAccountClaims.Subject, bundle.AppAccountJWT)
	if err != nil {
		panic(err)
	}
	//appAccount := server.NewAccount("AppAccount")
	//authCallout := &server.AuthCallout{
	//	Issuer:    bundle.AppAccountJWT,
	//	Account:   bundle.AppAccountJWT,
	//	AuthUsers: nil,
	//	XKey:      "",
	//}
	systemAccount := server.NewAccount("SYS")
	systemAccount.Nkey = systemAccountNKeyPub
	_ = systemAccount
	appAccount := server.NewAccount("AppAccount")
	appAccount.Nkey = appAccountNKeyPub

	//appAccount.Issuer = operatorNKeyPub
	//appAccount := &server.Account{
	//	Name:   "AppAccount",
	//	Nkey:   bundle.AppAccountJWT,
	//	Issuer: bundle.OperatorJWT,
	//}
	_ = appAccount

	_ = userNKeyPub

	//servicePub, _ := bundle.ServiceNKey.PublicKey()
	opts := &server.Options{
		Host:          "127.0.0.1", // must match the address on the generated cert
		Port:          9998,        // some random test port that doesn't interfere
		SystemAccount: "SYS",

		//AuthCallout: authCallout,

		JetStream: true,
		//TrustedOperators:   []*jwt.OperatorClaims{operatorClaims},
		JetStreamMaxMemory: 10 * 1024 * 1024,
		//NoAuthUser:         "unauthenticated",
		Accounts: []*server.Account{systemAccount, appAccount},
		Nkeys: []*server.NkeyUser{
			{
				Nkey:        serviceNKeyPub,
				Permissions: adminPermissions,
				Account:     appAccount,
				//SigningKey:             "",
				SigningKey: appSigningNKeyPub,
				//AllowedConnectionTypes: nil,
			},
			{
				Nkey:        userNKeyPub,
				Permissions: userPermissions,
				Account:     appAccount,
				SigningKey:  appSigningNKeyPub,
				//AllowedConnectionTypes: nil,
			},
		},

		Users: []*server.User{
			//{
			//	Username:    "unauthenticated",
			//	Password:    "",
			//	Permissions: defaultPermissions,
			//	Account:     appAccount,
			//},
			{
				Username:    "sys",
				Permissions: adminPermissions,
				Account:     systemAccount,
			},
			{
				Username:    bundle.UserID,
				Permissions: userPermissions,
				Account:     appAccount,
			},
			{
				Username:    "testuser1",
				Password:    "testpass1",
				Permissions: adminPermissions,
				Account:     appAccount,
			},
		},
	}

	if svr.caCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(svr.caCert)
		clientCertList := []tls.Certificate{*svr.serverCert}
		tlsConfig := &tls.Config{
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
		}
		opts.TLSConfig = tlsConfig
	}

	ns, err := server.NewServer(opts)

	if err != nil {
		return "", err
	}
	svr.ns = ns
	svr.account = svr.ns.GlobalAccount()
	go ns.Start()
	if !ns.ReadyForConnections(4 * time.Second) {
		panic("not ready for connection")
	}
	clientURL = ns.ClientURL()
	return clientURL, nil
}

func (svr *TestNKeyServer) Stop() {
	if svr.ns != nil {
		svr.ns.Shutdown()
	}
}

// NewTestNKeyServer create a new test server instance using static configuration
//
//	serverCert optional cert for 127.0.0.1
func NewTestNKeyServer(serverCert *tls.Certificate, caCert *x509.Certificate) *TestNKeyServer {
	ts := &TestNKeyServer{
		serverCert: serverCert,
		caCert:     caCert,
	}
	return ts
}
