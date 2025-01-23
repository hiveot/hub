package sseclient

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"log/slog"
	"time"
)

// SsescAgentClient extends the SsescConsumerClient with agent methods for
// receiving requests and sending notifications over SSE-SC return channel.
//
// This is based on the SsescConsumerClient.
type SsescAgentClient struct {
	SsescConsumerClient
	httpbasicclient.HttpAgentClient
}

// handle agent requests if any
func (cl *SsescAgentClient) handleAgentRequest(req transports.RequestMessage) {
	resp := cl.OnRequest(req)

	// send the response to the caller
	err := cl.SendResponse(resp)
	if err != nil {
		slog.Error("handleAgentRequest: failed", "err", err.Error())
	}
}

// Init Initializes the HTTP/SSE-SC agent client transport
//
//	fullURL full path of the sse endpoint
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func (cl *SsescAgentClient) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) {

	// forms are not used for agents
	cl.SsescConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, nil, timeout)
	cl.HttpAgentClient.Init(&cl.SsescConsumerClient.HttpConsumerClient)
	cl.agentRequestHandler = cl.handleAgentRequest
}

// NewSsescAgentClient creates a new instance of the agent client with SSE-SC
// return-channel.
//
//	fullURL full path of the sse endpoint
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewSsescAgentClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *SsescAgentClient {

	cl := SsescAgentClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, timeout)
	return &cl
}
