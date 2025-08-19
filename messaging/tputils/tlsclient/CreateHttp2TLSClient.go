package tlsclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"
)

// NewHttp2TLSClient creates a http client setup for http/2
func NewHttp2TLSClient(caCert *x509.Certificate, clientCert *tls.Certificate, timeout time.Duration) *http.Client {
	var clientCertList []tls.Certificate

	// the CA certificate is set in NewTLSClient
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	if clientCert != nil {
		clientCertList = []tls.Certificate{*clientCert}
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: caCert == nil,
		Certificates:       clientCertList,
	}
	//tlsTransport := http.DefaultTransport.(*http.Transport)
	tlsTransport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			c, err := tls.Dial(network, addr, cfg)
			return c, err
		},
		TLSClientConfig: tlsConfig,
	}
	// add a cookie jar for storing cookies
	cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	cjar, err := cookiejar.New(cjarOpts)
	if err != nil {
		err = fmt.Errorf("NewHttp2TLSClient: error creating cookiejar. Continuing anyways: %w", err)
		slog.Error(err.Error())
		err = nil
	}
	// Dont set a timeout here as it will end the connection
	httpClient := &http.Client{
		Transport: tlsTransport,
		Jar:       cjar,
	}
	return httpClient
}
