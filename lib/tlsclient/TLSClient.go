// Package tlsclient with a TLS client helper supporting certificate, JWT or Basic authentication
package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// TLSClient is a simple TLS Client with authentication using certificates or JWT authentication with login/pw
type TLSClient struct {
	// host and port of the server to setup to
	hostPort   string
	caCert     *x509.Certificate
	httpClient *http.Client
	timeout    time.Duration

	// client certificate mutual authentication
	clientCert *tls.Certificate

	// JWT bearer token after login, refresh, or external source
	// Invoke will use this if set.
	bearerToken string
}

// Certificate returns the client auth certificate or nil if none is used
func (cl *TLSClient) Certificate() *tls.Certificate {
	return cl.clientCert
}

// Close the connection with the server
func (cl *TLSClient) Close() {
	slog.Info("TLSClient.Close: Closing client connection")

	if cl.httpClient != nil {
		cl.httpClient.CloseIdleConnections()
		cl.httpClient = nil
	}
}

// ConnectWithBasicAuth creates a server connection using the configured authentication
// Intended to setup to services that do not support JWT authentication
//
//	func (cl *TLSClient) ConnectWithBasicAuth(userID string, passwd string) {
//		cl.clientID = userID
//		//cl.basicSecret = passwd
//		// Invoke() will use basic auth if basicSecret is set
//
//		cl.httpClient = cl.setup()
//	}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
func (cl *TLSClient) ConnectWithClientCert(clientCert *tls.Certificate) (err error) {
	if clientCert == nil {
		err = fmt.Errorf("TLSClient.ConnectWithClientCert, No client key/certificate provided")
		slog.Error(err.Error())
		return err
	}

	// test if the given cert is valid for our CA
	if cl.caCert != nil {
		caCertPool := x509.NewCertPool()
		if cl.caCert != nil {
			caCertPool.AddCert(cl.caCert)
		}
		opts := x509.VerifyOptions{
			Roots:     caCertPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
		if err == nil {
			// FIXME: TestCertAuth: certificate specifies incompatible key usage
			// why? Is the certpool invalid? Yet the test succeeds
			_, err = x509Cert.Verify(opts)
		}
		if err != nil {
			err = fmt.Errorf("ConnectWithClientCert: certificate verfication failed: %w", err)
			slog.Error(err.Error())
			return err
		}
	}
	cl.clientCert = clientCert
	cl.httpClient = CreateHttp2TLSClient(cl.caCert, clientCert, cl.timeout)

	return nil
}

// ConnectWithToken Sets login ID and secret for bearer token authentication using a
// token obtained at login or elsewhere.
//
// No error is returned as this just sets up the token and http client. No messages are send yet.
func (cl *TLSClient) ConnectWithToken(token string) {
	cl.bearerToken = token
	if cl.httpClient != nil {
		cl.httpClient = cl.setup()
	}
}

// Delete sends a delete message
// Note that delete methods do not allow a body, or a 405 is returned.
//
//	path to invoke
func (cl *TLSClient) Delete(path string) ([]byte, int, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("DELETE", serverURL, nil, nil)
}

// Get is a convenience function to send a request
//
//	path to invoke
func (cl *TLSClient) Get(path string) ([]byte, int, error) {
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("GET", serverURL, nil, nil)
}

// GetHttpClient returns the underlying HTTP client
func (cl *TLSClient) GetHttpClient() *http.Client {
	return cl.httpClient
}

// Invoke a HTTPS method and read response using content type application/json
// If a JWT authentication is enabled then add the bearer token to the header
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	method: GET, PUT, POST, ...
//	url: full URL to invoke
//	msg contains the request body as a string or object
//	qParams: optional map with query parameters
//
// This returns the response data, return status code or an error
func (cl *TLSClient) Invoke(method string, requrl string,
	msg interface{}, qParams map[string]string) ([]byte, int, error) {

	var body []byte
	var err error
	var req *http.Request
	contentType := "application/json"

	if cl == nil || cl.httpClient == nil {
		err = fmt.Errorf("Invoke: '%s'. Client is not started", requrl)
		return nil, http.StatusInternalServerError, err
	}
	slog.Debug("TLSClient.Invoke", "method", method, "requrl", requrl)

	// careful, a double // in the path causes a 301 and changes post to get
	// requrl := fmt.Sprintf("https://%s%s", hostPort, path)
	if msg != nil {
		// only marshal to JSON if this isn't a string
		switch msgWithType := msg.(type) {
		case string:
			body = []byte(msgWithType)
		case []byte:
			body = msgWithType
		default:
			body, _ = json.Marshal(msg)
		}
	}
	req, err = NewRequest(method, requrl, cl.bearerToken, body)
	// optional query parameters
	if qParams != nil {
		qValues := req.URL.Query()
		for k, v := range qParams {
			qValues.Add(k, v)
		}
		req.URL.RawQuery = qValues.Encode()
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// set headers
	req.Header.Set("Content-Type", contentType)

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Invoke: %s %s: %w", method, requrl, err)
		slog.Error(err.Error())
		return nil, 500, err
	}
	respBody, err := io.ReadAll(resp.Body)
	// response body MUST be closed
	_ = resp.Body.Close()

	// FIXME: detect difference between connect and unauthenticated

	if resp.StatusCode == 401 {
		err = fmt.Errorf("%s", resp.Status)
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		err = fmt.Errorf("%s: %s", resp.Status, respBody)
		if resp.Status == "" {
			err = fmt.Errorf("%d (%s): %s", resp.StatusCode, resp.Status, respBody)
		}
	} else if resp.StatusCode >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", resp.StatusCode, resp.Status, respBody)
		slog.Error("Invoke returned internal server error", "requrl", requrl, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("Invoke: Error %s %s: %w", method, requrl, err)
	}
	return respBody, resp.StatusCode, err
}

//// Logout from the server and end the session
//func (cl *TLSClient) Logout() error {
//	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostLogoutPath)
//	_, err := cl.Invoke("POST", serverURL, http.NoBody, nil)
//	return err
//}

// Patch sends a patch message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Patch(path string, msg interface{}) ([]byte, int, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PATCH", serverURL, msg, nil)
}

// Post a message.
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Post(path string, msg interface{}) ([]byte, int, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("POST", serverURL, msg, nil)
}

// Put a message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Put(path string, msg interface{}) ([]byte, int, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PUT", serverURL, msg, nil)
}

// setup sets-up the http client with TLS transport
func (cl *TLSClient) setup() *http.Client {
	//// the CA certificate is set in NewTLSClient
	httpClient := CreateHttp2TLSClient(cl.caCert, nil, cl.timeout)

	// FIXME:
	// 1 does this work if the server is connected using an IP address?
	// 2. How are cookies stored between sessions?
	cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	cjar, err := cookiejar.New(cjarOpts)
	if err != nil {
		err = fmt.Errorf("NewTLSClient: error setting cookiejar. The use of bearer tokens might not work: %w", err)
		slog.Error(err.Error())
	}
	httpClient.Jar = cjar
	return httpClient
}

//// SetBearerToken sets the authentication token for the http header
//func (cl *TLSClient) SetBearerToken(token string) {
//	cl.bearerToken = token
//}

// NewTLSClient creates a new TLS Client instance.
// Use setup/Close to open and close connections
//
//	hostPort is the server hostname or IP address and port to setup to
//	caCert with the x509 CA certificate, nil if not available
//	timeout duration of the request or 0 for default of 10 seconds
//
// returns TLS client for submitting requests
func NewTLSClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) *TLSClient {
	if timeout == 0 {
		timeout = time.Second * 30
	}

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
	}

	cl := &TLSClient{
		hostPort: hostPort,
		timeout:  timeout,
		caCert:   caCert,
	}

	// setup the connection
	cl.httpClient = cl.setup()
	return cl
}
