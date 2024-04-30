// Package tlsclient with a TLS client helper supporting certificate, JWT or Basic authentication
package tlsclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"golang.org/x/net/publicsuffix"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// Authentication methods for use with ConnectWithLoginID
// Use AuthMethodDefault unless there is a good reason not to
const (
	AuthMethodBasic  = "basic"  // basic auth for backwards compatibility when connecting
	AuthMethodDigest = "digest" // digest auth for backwards compatibility when connecting
	AuthMethodNone   = ""       // disable authentication, for testing
	AuthMethodJwt    = "jwt"    // JSON web token for use with WoST server (default)
)

// standardized query parameter names for querying servers
const (
	// ParamOffset offset in case of multiple requests
	ParamOffset = "offset"
	// ParamLimit contains maximum number of results
	ParamLimit = "limit"
	// ParamQuery contains a query
	ParamQuery = "queryparams"
	// ParamUpdatedSince contains a ISO8601 datetime
	ParamUpdatedSince = "updatedSince"
	// ParamThings contains a list of Thing IDs to query for
	ParamThings = "things"
)

// TLSClient is a simple TLS Client with authentication using certificates or JWT authentication with login/pw
type TLSClient struct {
	// host and port of the server to connect to
	hostPort        string
	loginPath       string // path on server to login
	refreshPath     string // path on server to refresh token
	caCert          *x509.Certificate
	caCertPool      *x509.CertPool
	httpClient      *http.Client
	timeout         time.Duration
	checkServerCert bool

	// client certificate mutual authentication
	clientCert *tls.Certificate

	// User ID for authentication
	clientID string

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

// connect sets-up the http client with TLS transport
func (cl *TLSClient) connect() *http.Client {
	// the CA certificate is set in NewTLSClient
	tlsConfig := &tls.Config{
		RootCAs:            cl.caCertPool,
		InsecureSkipVerify: !cl.checkServerCert,
	}

	tlsTransport := http.DefaultTransport
	tlsTransport.(*http.Transport).TLSClientConfig = tlsConfig

	// FIXME:
	// 1 does this work if the server is connected using an IP address?
	// 2. How are cookies stored between sessions?
	cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	cjar, err := cookiejar.New(cjarOpts)
	if err != nil {
		err = fmt.Errorf("NewTLSClient: error setting cookiejar. The use of bearer tokens might not work: %w", err)
		slog.Error(err.Error())
	}

	return &http.Client{
		Transport: tlsTransport,
		Timeout:   cl.timeout,
		Jar:       cjar,
	}
}

// ConnectNoAuth creates a connection with the server without client authentication
// Only requests that do not require authentication will succeed
func (cl *TLSClient) ConnectNoAuth() {
	cl.httpClient = cl.connect()
}

// ConnectWithBasicAuth creates a server connection using the configured authentication
// Intended to connect to services that do not support JWT authentication
//func (cl *TLSClient) ConnectWithBasicAuth(userID string, passwd string) {
//	cl.clientID = userID
//	//cl.basicSecret = passwd
//	// Invoke() will use basic auth if basicSecret is set
//
//	cl.httpClient = cl.connect()
//}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
func (cl *TLSClient) ConnectWithClientCert(clientCert *tls.Certificate) (err error) {
	var clientCertList = make([]tls.Certificate, 0)

	if clientCert == nil {
		err = fmt.Errorf("TLSClient.ConnectWithClientCert, No client key/certificate provided")
		slog.Error(err.Error())
		return err
	}

	// test if the given cert is valid for our CA
	if cl.caCert != nil {
		opts := x509.VerifyOptions{
			Roots:     cl.caCertPool,
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
	clientCertList = append(clientCertList, *clientCert)

	tlsConfig := &tls.Config{
		RootCAs:            cl.caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: !cl.checkServerCert,
	}

	tlsTransport := http.DefaultTransport
	tlsTransport.(*http.Transport).TLSClientConfig = tlsConfig

	cl.httpClient = &http.Client{
		Transport: tlsTransport,
		Timeout:   cl.timeout,
	}
	return nil
}

// ConnectWithToken Sets login ID and secret for JWT authentication using a
// token obtained at login or elsewhere.
//
// No error is returned as this just sets up the token and http client. No messages are send yet.
func (cl *TLSClient) ConnectWithToken(loginID string, token string) {
	cl.clientID = loginID
	cl.bearerToken = token

	cl.httpClient = cl.connect()
}

// ConnectWithPassword requests JWT tokens using loginID/password
// If a CA certificate is not available then insecure-skip-verify is used to allow
// connection to an unverified server (leap of faith).
//
// The server returns a JwtAuthResponse message with a jwt bearer token for use in the authorization header.
//
//	loginID username or application ID to identify as.
//	secret to authenticate with.
//
// Returns new auth token if successful or an error if setting up of authentication failed.
func (cl *TLSClient) ConnectWithPassword(loginID string, secret string) (token string, err error) {
	cl.clientID = loginID

	loginURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.loginPath)

	// create tlsTransport
	cl.httpClient = cl.connect()

	loginMessage := api.LoginArgs{
		ClientID: loginID,
		Password: secret,
	}
	resp, err2 := cl.Invoke("POST", loginURL, loginMessage)
	if err2 != nil {
		err = fmt.Errorf("ConnectWithPassword: login to %s failed. %s", loginURL, err2)
		return "", err
	}
	reply := api.LoginResp{}
	err = json.Unmarshal(resp, &reply)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		return "", err
	}
	cl.bearerToken = reply.Token
	return cl.bearerToken, err
}

// Delete sends a delete message
// Note that delete methods do not allow a body, or a 405 is returned.
//
//	path to invoke
func (cl *TLSClient) Delete(path string) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("DELETE", url, nil)
}

// Get is a convenience function to send a request
//
//	path to invoke
func (cl *TLSClient) Get(path string) ([]byte, error) {
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("GET", url, nil)
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
func (cl *TLSClient) Invoke(method string, url string, msg interface{}) ([]byte, error) {
	var body []byte
	var err error
	var req *http.Request
	contentType := "application/json"

	if cl == nil || cl.httpClient == nil {
		err = fmt.Errorf("Invoke: '%s'. Client is not started", url)
		return nil, err
	}
	slog.Info("TLSClient.Invoke", "method", method, "url", url)

	// careful, a double // in the path causes a 301 and changes post to get
	// url := fmt.Sprintf("https://%s%s", hostPort, path)
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
	req, err = cl.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("Content-Type", contentType)

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Invoke: %s %s: %w", method, url, err)
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("%s: %s", resp.Status, respBody)
		if resp.Status == "" {
			msg = fmt.Sprintf("%d (%s): %s", resp.StatusCode, resp.Status, respBody)
		}
		err = errors.New(msg)
	}
	if err != nil {
		err = fmt.Errorf("Invoke: Error %s %s: %w", method, url, err)
		return nil, err
	}
	return respBody, err
}

// NewRequest creates a request object containing a bearer token if available
func (cl *TLSClient) NewRequest(method string, fullURL string, body []byte) (*http.Request, error) {
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}
	// set the intended destination
	// in web browser this is the origin that provided the web page,
	// here it means the server that we'd like to talk to.
	parts, err := url.Parse(fullURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Set("Origin", origin)
	if cl.bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+cl.bearerToken)
	} else {
		// no authentication
	}
	return req, nil
}

// Logout from the server and end the session
func (cl *TLSClient) Logout() error {
	url := fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostLogoutPath)
	_, err := cl.Invoke("POST", url, http.NoBody)
	return err
}

// Post a message.
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Post(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("POST", url, msg)
}

// Put a message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Put(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PUT", url, msg)
}

// Patch sends a patch message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Patch(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PATCH", url, msg)
}

// RefreshToken refreshes the JWT token
//
// refreshURL is an optional alternative URL, or "" for the default path
//
// This returns a struct with new access and refresh token
func (cl *TLSClient) RefreshToken(refreshURL string) (newToken string, err error) {
	if refreshURL == "" {
		refreshURL = fmt.Sprintf("https://%s%s", cl.hostPort, cl.refreshPath)
	}
	args := api.RefreshTokenArgs{
		OldToken: cl.bearerToken, ClientID: cl.clientID,
	}
	data, _ := json.Marshal(args)
	// the bearer token holds the old token
	resp, err := cl.Invoke("POST", refreshURL, data)
	// set the new token as the bearer token
	if err == nil {
		reply := api.RefreshTokenResp{}
		err = json.Unmarshal(resp, &reply)

		if err == nil {
			newToken = reply.Token
			cl.bearerToken = newToken
		}
	}
	// old token exists in client cookie
	//req, err := http.NewRequest("POST", refreshURL, http.NoBody)
	//var resp *http.Response
	//if err != nil {
	//	err = fmt.Errorf("RefreshToken: Error creating request for URL %s: %w", refreshURL, err)
	//	return "", err
	//}
	//req.Header.Set("Content-Type", "application/json")
	//resp, err = cl.httpClient.Do(req)
	//
	//if err != nil {
	//	err = fmt.Errorf("RefreshToken: Error using URL %s: %w", refreshURL, err)
	//	return "", err
	//} else if resp.StatusCode >= 400 {
	//	err = fmt.Errorf("RefreshToken: refresh using URL %s failed with: %s", refreshURL, resp.Status)
	//	return "", err
	//}
	//respBody, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	err = fmt.Errorf("RefreshToken: failed with error %w", err)
	//	return "", err
	//}
	//err = json.Unmarshal(respBody, &newToken)
	return newToken, err
}

// NewTLSClient creates a new TLS Client instance.
// Use connect/Close to open and close connections
//
//	hostPort is the server hostname or IP address and port to connect to
//	caCert with the x509 CA certificate, nil if not available
//	timeout duration of the request or 0 for default of 10 seconds
//
// returns TLS client for submitting requests
func NewTLSClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) *TLSClient {
	var checkServerCert bool
	caCertPool := x509.NewCertPool()
	if timeout == 0 {
		timeout = time.Second * 10
	}

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
		checkServerCert = false
	} else {
		slog.Info("NewTLSClient: CA certificate",
			slog.String("destination", hostPort),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
		checkServerCert = true
	}

	cl := &TLSClient{
		hostPort:        hostPort,
		loginPath:       vocab.PostLoginPath,
		refreshPath:     vocab.PostRefreshPath,
		timeout:         timeout,
		caCertPool:      caCertPool,
		caCert:          caCert,
		checkServerCert: checkServerCert,
	}

	return cl
}
