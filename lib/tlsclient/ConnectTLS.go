package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
)

// ConnectTLS creates a TLS connection to a server, optionally using a client certificate.
//
//	serverURL full URL:  tls://host:8883, ssl://host:8883,  wss://host:9001, tcps://awshost:8883/mqtt
//	clientCert to login with. Nil to not use client certs
//	caCert of the server to connect to (recommended). Nil to not verify the server connection.
func ConnectTLS(serverURL string, clientCert *tls.Certificate, caCert *x509.Certificate) (
	*tls.Conn, error) {

	// connect always uses TLS
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
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	broker, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	conn, err := tls.Dial(broker.Scheme, broker.Host, tlsConfig)
	return conn, err
}
