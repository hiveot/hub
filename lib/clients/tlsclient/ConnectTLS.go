package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"strings"
)

// ConnectTLS creates a TLS connection to a server, optionally using a client certificate.
//
//	serverURL full URL:  tls/tcp/tcps://host:8883,  wss://host:9001
//	clientCert to login with. Nil to not use client certs
//	caCert of the server to setup to (recommended). Nil to not verify the server connection.
func ConnectTLS(serverURL string, clientCert *tls.Certificate, caCert *x509.Certificate) (
	*tls.Conn, error) {

	// setup always uses TLS
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsOpts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	// if a client certificate is given then include it
	clientCertList := make([]tls.Certificate, 0)
	if clientCert != nil {
		x509Cert, _ := x509.ParseCertificate(clientCert.Certificate[0])
		clientCertList = append(clientCertList, *clientCert)
		_, err := x509Cert.Verify(tlsOpts)
		if err != nil {
			return nil, err
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	// url.Parse doesn't handle in memory UDS address starting with 'unix://@/path'
	broker, err := url.Parse(serverURL)
	if strings.HasPrefix(serverURL, "unix://@") {
		broker.Scheme = "unix"
		broker.Path = serverURL[7:]
	}

	if err != nil {
		return nil, err
	}
	// if no scheme is given then its a unix path
	if broker.Scheme == "" {
		broker.Scheme = "unix"
	}
	if broker.Host == "" {
		broker.Host = broker.Path
	}
	// dial doesn't support tcps or mqtts
	if broker.Scheme == "tcps" || broker.Scheme == "mqtts" || broker.Scheme == "tls" {
		broker.Scheme = "tcp"
	}
	conn, err := tls.Dial(broker.Scheme, broker.Host, tlsConfig)
	return conn, err
}
