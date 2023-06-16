package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nats-io/nats-server/v2/server"
	"time"
)

// TestServer is an embedded NATS test messaging server
type TestServer struct {
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	ns         *server.Server
}

// Start the server, listening on 127.0.0.1
func (svr *TestServer) Start() (clientURL string, err error) {
	opts := &server.Options{Host: "127.0.0.1"}

	if svr.caCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(svr.caCert)
		clientCertList := []tls.Certificate{*svr.serverCert}
		tlsConfig := &tls.Config{
			RootCAs:      caCertPool,
			Certificates: clientCertList,
		}
		opts.TLSConfig = tlsConfig
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return "", err
	}
	svr.ns = ns
	go ns.Start()
	if !ns.ReadyForConnections(4 * time.Second) {
		panic("not ready for connection")
	}
	clientURL = ns.ClientURL()
	return clientURL, nil
}

func (svr *TestServer) Stop() {
	if svr.ns != nil {
		svr.ns.Shutdown()
	}
}

// NewTestServer create a new test server instance
//
//	serverCert optional cert for 127.0.0.1
func NewTestServer(serverCert *tls.Certificate, caCert *x509.Certificate) *TestServer {
	ts := &TestServer{
		serverCert: serverCert,
		caCert:     caCert,
	}
	return ts
}
