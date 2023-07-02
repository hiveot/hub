package nats

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/nats-io/nats-server/v2/server"
	"time"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}, Deny: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>"}, Deny: []string{">"}},
}

var adminPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
}

// HubNats is the adapter for configuring the NATS server.
// This configures the server for publishing  provides a static configuration for the server for authn, authz, directory, and history streaming
type HubNats struct {
	hubAccount *server.Account
	serverOpts *server.Options
	ns         *server.Server
}

// Start the Hub and NATS servers
// This configures an admin user
//
//	host to listen on or "" for all interfaces
//	port to listen or 0 for default port
//	serverCert is the CA signed server certificate for the given host
//	caCert is the certificate of the CA that signed the server crt
func (srv *HubNats) Start(host string, port int, serverCert *tls.Certificate, caCert *x509.Certificate) (clientURL string, err error) {
	if host == "" {
		host = "127.0.0.1"
	}

	//adminPermissions := &server.Permissions{
	//	Publish:   &server.SubjectPermission{Allow: []string{">"}},
	//	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
	//}
	//pass1Bcrypted, _ := bcrypt.GenerateFromPassword([]byte("pass1"), 14)
	//user1NKey, _ := nkeys.CreateUser()

	srv.serverOpts = &server.Options{
		//Accounts:           []*server.Account{srv.hubAccount},
		Host:               host,
		Port:               port,
		JetStream:          true,
		JetStreamMaxMemory: 1 * 1024 * 1024 * 1024,
		//Accounts:           []*server.Account{hubAccount},
		//TLSMap: true,   // this requires client cert for ALL clients :(
		//LogFile:    "/tmp/nats.log",
		//Debug:      true,
		//Trace:      true,
		NoAuthUser: "unauthenticated",
		Nkeys:      []*server.NkeyUser{},
		Users: []*server.User{
			{
				Username:    "unauthenticated",
				Password:    "",
				Permissions: defaultPermissions,
			}, {
				Username:    "admin",
				Password:    "pass1",
				Permissions: adminPermissions,
			},
		},
	}

	if caCert != nil {
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
		srv.serverOpts.TLSConfig = tlsConfig
	}
	// start nats
	srv.ns, err = server.NewServer(srv.serverOpts)
	srv.ns.ConfigureLogger()
	if err == nil {
		srv.hubAccount = srv.ns.GlobalAccount()
	}
	if err == nil {
		go srv.ns.Start()
		if !srv.ns.ReadyForConnections(3 * time.Second) {
			err = errors.New("nats: not ready for connection")
		} else {
			clientURL = srv.ns.ClientURL()
		}
	}
	//srv.ns.Reload()
	// start JetStream
	//if err == nil {
	//	jsConfig := server.JetStreamConfig{
	//		MaxMemory:  1 * 1024 * 1024 * 1024,
	//		MaxStore:   0,
	//		StoreDir:   "/tmp/hiveot/store",
	//		Domain:     "",
	//		CompressOK: false,
	//	}
	//	err = srv.ns.EnableJetStream(&jsConfig)
	//}
	if err == nil {
		// start discovery
	}

	return clientURL, err
}

// Stop the server
func (srv *HubNats) Stop() {
	srv.ns.Shutdown()
}
