// Package tlsclient with a TLS client helper supporting certificate, JWT or Basic authentication
package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const DefaultClientTimeout = time.Second * 30

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

	// headers to include in each request
	headers map[string]string
}

// Certificate returns the client auth certificate or nil if none is used
func (cl *TLSClient) Certificate() *tls.Certificate {
	return cl.clientCert
}

// Close the connection with the server
func (cl *TLSClient) Close() {
	slog.Debug("TLSClient.Remove: Closing client connection")

	if cl.httpClient != nil {
		cl.httpClient.CloseIdleConnections()
		//cl.httpClient = nil
	}
}

// Delete sends a delete message
// Note that delete methods do not allow a body, or a 405 is returned.
//
//	path to invoke
func (cl *TLSClient) Delete(path string) (resp []byte, httpStatus int, err error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	resp, httpStatus, _, err = cl.Send("DELETE", serverURL, nil, "", nil)
	return resp, httpStatus, err

}

// Get is a convenience function to send a request
// This returns the response data, the http status code and an error of delivery failed
//
//	path to invoke
func (cl *TLSClient) Get(path string) (resp []byte, httpStatus int, err error) {
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	resp, httpStatus, _, err = cl.Send("GET", serverURL, nil, "", nil)
	return resp, httpStatus, err
}

// GetHttpClient returns the underlying HTTP client
func (cl *TLSClient) GetHttpClient() *http.Client {
	return cl.httpClient
}

// Send a HTTPS method and read response.
//
// If a JWT authentication is enabled then add the bearer token to the header
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	method: GET, PUT, POST, ...
//	url: full URL to invoke
//	body contains the serialized request body
//	contentType: default is "application/json"
//	qParams: optional map with query parameters
//
// This returns the serialized response data, a response message ID, return status code or an error
func (cl *TLSClient) Send(
	method string, requrl string, body []byte, contentType string, qParams map[string]string) (
	resp []byte, httpStatus int, headers http.Header, err error) {

	var req *http.Request
	if contentType == "" {
		contentType = "application/json"
	}
	if cl == nil || cl.httpClient == nil {
		err = fmt.Errorf("_send: '%s'. Client is not started", requrl)
		return nil, http.StatusInternalServerError, nil, err
	}
	slog.Debug("TLSClient.Send", "method", method, "requrl", requrl)

	// Caution! a double // in the path causes a 301 and changes post to get
	req, err = NewRequest(method, requrl, cl.bearerToken, body)
	// optional query parameters
	if err == nil && qParams != nil {
		qValues := req.URL.Query()
		for k, v := range qParams {
			qValues.Add(k, v)
		}
		req.URL.RawQuery = qValues.Encode()
	}
	if err != nil {
		return nil, http.StatusInternalServerError, nil, err
	}

	// set headers
	req.Header.Set("Content-Type", contentType)
	for k, v := range cl.headers {
		req.Header.Set(k, v)
	}

	httpResp, err := cl.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Send: %s %s: %w", method, requrl, err)
		slog.Error(err.Error())
		return nil, 500, nil, err
	} else if httpResp.StatusCode >= 300 {
		err = fmt.Errorf("Send: %s %s: failed with (%d) %s",
			method, requrl, httpResp.StatusCode, httpResp.Status)
		slog.Error(err.Error())
		return nil, httpResp.StatusCode, nil, err
	}
	respBody, err := io.ReadAll(httpResp.Body)
	// response body MUST be closed
	_ = httpResp.Body.Close()
	httpStatus = httpResp.StatusCode

	if httpStatus == 401 {
		err = fmt.Errorf("%s", httpResp.Status)
	} else if httpStatus >= 400 && httpStatus < 500 {
		err = fmt.Errorf("%s: %s", httpResp.Status, respBody)
		if httpResp.Status == "" {
			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
		}
	} else if httpStatus >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
		slog.Error("Send returned internal server error", "requrl", requrl, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("Send: Error %s %s: %w", method, requrl, err)
	}
	return respBody, httpStatus, httpResp.Header, err
}

//// Logout from the server and end the session
//func (cl *TLSClient) Logout() error {
//	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostLogoutPath)
//	_, err := cl._send("POST", serverURL, http.NoBody, nil)
//	return err
//}

// Patch sends a patch message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized body
func (cl *TLSClient) Patch(
	path string, body []byte) (resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	resp, statusCode, _, err = cl.Send(http.MethodPatch, serverURL, body, "", nil)
	return resp, statusCode, err
}

// Post a message.
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized request body
func (cl *TLSClient) Post(path string, body []byte) (
	resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	resp, statusCode, _, err = cl.Send(
		http.MethodPost, serverURL, body, "", nil)
	return resp, statusCode, err
}

// PostForm posts a form message.
func (cl *TLSClient) PostForm(path string, formData map[string]string) (
	resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	form := url.Values{}
	for k, v := range formData {
		form.Add(k, v)
	}
	body := form.Encode()
	resp, statusCode, _, err = cl.Send(
		http.MethodPost, serverURL, []byte(body), "application/x-www-form-urlencoded", nil)
	return resp, statusCode, err
}

// Put a message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized request body
//	correlationID optional field to link async requests and responses
func (cl *TLSClient) Put(path string, body []byte) (
	resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	resp, statusCode, _, err = cl.Send(http.MethodPut, serverURL, body, "", nil)
	return resp, statusCode, err
}

// SetAuthToken Sets login ID and secret for bearer token authentication using a
// token obtained at login or elsewhere.
//
// No error is returned as this just sets up the token for future use. No messages are send yet.
func (cl *TLSClient) SetAuthToken(token string) {
	cl.bearerToken = token
}

// SetHeader sets a header to include in each request
// use an empty value to remove the header
func (cl *TLSClient) SetHeader(name string, val string) {
	if val == "" {
		delete(cl.headers, name)
	} else {
		cl.headers[name] = val
	}
}

//// SetBearerToken sets the authentication token for the http header
//func (cl *TLSClient) SetBearerToken(token string) {
//	cl.bearerToken = token
//}

// NewTLSClient creates a new TLS Client instance.
// Use setup/Remove to open and close connections
//
//	hostPort is the server hostname or IP address and port to setup to
//	clientCert is an option client certificate used to connect
//	caCert with the x509 CA certificate, nil if not available
//	timeout duration of the request or 0 for default
//
// returns TLS client for submitting requests
func NewTLSClient(hostPort string, clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *TLSClient {

	if timeout == 0 {
		timeout = DefaultClientTimeout
	}

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
	}

	// cert verification
	// if a client cert is given then test if it is valid for our CA.
	// this detects problems with certs that can be hard to track down
	if caCert != nil && clientCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(caCert)

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
			err = fmt.Errorf("NewTLSClient: certificate verfication failed: %w. Continuing for now.", err)
			slog.Error(err.Error())
		}
	}
	// create the client
	httpClient := NewHttp2TLSClient(caCert, clientCert, timeout)
	// this has moved to NewHttp2TlsClient
	//// add a cookie jar for storing cookies
	//cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	//cjar, err := cookiejar.New(cjarOpts)
	//if err != nil {
	//	err = fmt.Errorf("NewTLSClient: error setting cookiejar. The use of auth cookie might not persist. Continuing: %w", err)
	//	slog.Error(err.Error())
	//	err = nil
	//}
	//httpClient.Jar = cjar

	cl := &TLSClient{
		hostPort:   hostPort,
		httpClient: httpClient,
		timeout:    timeout,
		clientCert: clientCert,
		caCert:     caCert,
		headers:    make(map[string]string),
	}

	return cl
}
