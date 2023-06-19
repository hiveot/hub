package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// HubServer contains the embedded NATS JetStream server
// * custom authentication
// * custom authorization
type HubServer struct {
	serverCert *tls.Certificate
	caCert     *x509.Certificate
	ns         *server.Server
	js         nats.JetStream
}

func (hs *HubServer) AddUser(userID string, password string) {

}
func (hs *HubServer) RemoveUser(userID string) {

}
func (hs *HubServer) ListUsers() (u server.User) {
	return
}

// Start the Hub and NATS servers
func (hs *HubServer) Start() (clientURL string, err error) {
	//hubAccount := &server.Account{
	//	Name:   "hiveot",
	//	Nkey:   "",
	//	Issuer: "",
	//}
	adminPermissions := &server.Permissions{
		Publish:   &server.SubjectPermission{Allow: []string{">"}},
		Subscribe: &server.SubjectPermission{Allow: []string{">"}},
	}
	pass1Bcrypted, _ := bcrypt.GenerateFromPassword([]byte("pass1"), 14)

	serverOpts := &server.Options{
		Host:               "127.0.0.1",
		JetStream:          true,
		JetStreamMaxMemory: 1 * 1024 * 1024 * 1024,
		//Accounts:           []*server.Account{hubAccount},
		//TLSMap:     true,
		//LogFile:    "/tmp/nats.log",
		Debug:      true,
		Trace:      true,
		NoAuthUser: "unauthenticated",
		//Username:   "admin",
		//Password:   "pass1",
		Users: []*server.User{
			{
				Username:    "unauthenticated",
				Password:    "",
				Permissions: nil,
			},
			{
				Username: "admin",
				//Password:    string(pass1Bcrypted),
				Permissions: adminPermissions,
			},
			{
				Username:    "user1",
				Password:    string(pass1Bcrypted),
				Permissions: adminPermissions,
			},
		},
	}

	if hs.caCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(hs.caCert)
		clientCertList := []tls.Certificate{*hs.serverCert}
		tlsConfig := &tls.Config{
			ServerName:   "HiveOT Hub",
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}
		serverOpts.TLSConfig = tlsConfig
	}
	// start nats
	hs.ns, err = server.NewServer(serverOpts)
	hs.ns.ConfigureLogger()
	if err == nil {
		go hs.ns.Start()
		if !hs.ns.ReadyForConnections(3 * time.Second) {
			err = errors.New("nats: not ready for connection")
		} else {
			clientURL = hs.ns.ClientURL()
		}
	}
	//hs.ns.Reload()
	// start JetStream
	//if err == nil {
	//	jsConfig := server.JetStreamConfig{
	//		MaxMemory:  1 * 1024 * 1024 * 1024,
	//		MaxStore:   0,
	//		StoreDir:   "/tmp/hiveot/store",
	//		Domain:     "",
	//		CompressOK: false,
	//	}
	//	err = hs.ns.EnableJetStream(&jsConfig)
	//}
	if err == nil {
		// start discovery
	}
	return clientURL, err
}

// Stop the server
func (hs *HubServer) Stop() {
}

// NewHubServer creates a server instance using a certificate
func NewHubServer(serverCert *tls.Certificate, caCert *x509.Certificate) *HubServer {
	hs := &HubServer{
		serverCert: serverCert,
		caCert:     caCert,
	}
	return hs
}
