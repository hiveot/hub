package test

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/cells/transport/tlsclient"
)

// SSE client for testing
type SSEWebClient struct {
	connectHandler func(isConnected bool)
	msgHandler     func(evType string, data []byte)
	timeout        time.Duration

	tlsClient tlsclient.ITLSClient
}

// Close the connection with the server
func (cl *SSEWebClient) Close() {
	cl.tlsClient.Close()
	cl.connectHandler(false)
}

// Establish an sse connection to the web server
func (sseCl *SSEWebClient) Connect(clientID string, authToken string) error {
	err := sseCl.tlsClient.SetAuthToken(clientID, authToken)
	return err
}

// Send a http request to the web server
func (sseCl *SSEWebClient) Send(httpMethod string, path string, jsonData []byte) (
	body []byte, status int, err error) {

	resp, httpStatus, _, err := sseCl.tlsClient.Send(
		context.Background(), httpMethod, path, nil, jsonData, "")
	return resp, httpStatus, err
}

func (sseCl *SSEWebClient) SetAuthToken(clientID string, token string) error {
	return sseCl.tlsClient.SetAuthToken(clientID, token)
}

func (sseCl *SSEWebClient) SetTimeout(d time.Duration) {
	sseCl.timeout = d
}

func NewSSEWebClient(sseURL string, rootCAs *x509.CertPool,
	connectHandler func(isConnected bool),
	msgHandler func(evType string, data []byte)) *SSEWebClient {

	tlsClient := tlsclient.NewTLSClient(sseURL, rootCAs)

	webCl := &SSEWebClient{
		connectHandler: connectHandler,
		msgHandler:     msgHandler,
		tlsClient:      tlsClient,
	}
	return webCl
}
